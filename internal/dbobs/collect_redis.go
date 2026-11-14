package dbobs

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// CollectRedis reads INFO, DBSIZE, and SLOWLOG over RESP without external deps.
func CollectRedis(ctx context.Context, target Target, samples *[]QuerySample) (Metrics, error) {
	var m Metrics
	address := net.JoinHostPort(target.Host, strconv.Itoa(target.Port))
	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return m, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	reader := bufio.NewReader(conn)
	if target.Password != "" {
		if _, err := redisCommand(conn, reader, "AUTH", target.Password); err != nil {
			return m, fmt.Errorf("auth: %w", err)
		}
	}
	info, err := redisCommand(conn, reader, "INFO", "ALL")
	if err != nil {
		return m, err
	}
	values := map[string]string{}
	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if key, value, ok := strings.Cut(line, ":"); ok {
			values[key] = value
		}
	}

	m.Up = true
	m.Version = "Redis " + values["redis_version"]
	m.UptimeSec = parseFloat(values["uptime_in_seconds"])
	m.ConnsTotal = int(parseFloat(values["connected_clients"]))
	m.ConnsMax = int(parseFloat(values["maxclients"]))
	m.Blocked = int(parseFloat(values["blocked_clients"]))
	m.DBSizeBytes = int64(parseFloat(values["used_memory"]))
	if m.ConnsMax > 0 {
		m.ConnUsage = float64(m.ConnsTotal) / float64(m.ConnsMax) * 100
	}
	hits := parseFloat(values["keyspace_hits"])
	misses := parseFloat(values["keyspace_misses"])
	if hits+misses > 0 {
		m.CacheHitRatio = hits / (hits + misses) * 100
	}
	m.QueriesTotal = int64(parseFloat(values["total_commands_processed"]))
	m.SlowQueries = int(parseFloat(values["slowlog_len"]))
	m.IdleInTx = 0
	m.WaitingLocks = m.Blocked
	m.TempBytes = 0
	m.WalBytes = 0
	m.Deadlocks = 0
	m.Details = map[string]any{
		"ops_per_sec":       parseFloat(values["instantaneous_ops_per_sec"]),
		"keys":              parseRedisKeys(values["db0"]),
		"evicted_keys":      parseFloat(values["evicted_keys"]),
		"expired_keys":       parseFloat(values["expired_keys"]),
		"rejected_conns":     parseFloat(values["rejected_connections"]),
		"mem_fragmentation":  parseFloat(values["mem_fragmentation_ratio"]),
		"maxmemory":          parseFloat(values["maxmemory"]),
		"role":               values["role"],
	}

	slowlog, err := redisCommand(conn, reader, "SLOWLOG", "GET", "5")
	if err == nil {
		for _, entry := range parseRedisSlowlog(slowlog) {
			*samples = append(*samples, entry)
		}
	}
	return m, nil
}

// redisCommand sends a command and returns its bulk-string reply.
func redisCommand(conn net.Conn, reader *bufio.Reader, args ...string) (string, error) {
	var builder strings.Builder
	fmt.Fprintf(&builder, "*%d\r\n", len(args))
	for _, arg := range args {
		fmt.Fprintf(&builder, "$%d\r\n%s\r\n", len(arg), arg)
	}
	if _, err := conn.Write([]byte(builder.String())); err != nil {
		return "", err
	}
	return readRedisReply(reader, 0)
}

func readRedisReply(reader *bufio.Reader, depth int) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return "", nil
	}
	switch line[0] {
	case '+', ':':
		return line[1:], nil
	case '-':
		return "", fmt.Errorf("redis error: %s", line[1:])
	case '$':
		size, _ := strconv.Atoi(line[1:])
		if size < 0 {
			return "", nil
		}
		buf := make([]byte, size+2)
		if _, err := ioReadFull(reader, buf); err != nil {
			return "", err
		}
		return string(buf[:size]), nil
	case '*':
		count, _ := strconv.Atoi(line[1:])
		var parts []string
		for i := 0; i < count; i++ {
			part, err := readRedisReply(reader, depth+1)
			if err != nil {
				return "", err
			}
			parts = append(parts, part)
		}
		return strings.Join(parts, "|"), nil
	}
	return "", nil
}

func ioReadFull(reader *bufio.Reader, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := reader.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func parseRedisSlowlog(payload string) []QuerySample {
	// SLOWLOG GET flattening: entries are "id|timestamp|duration|args|client|name".
	fields := strings.Split(payload, "|")
	var samples []QuerySample
	for i := 0; i+3 < len(fields); i += 6 {
		durationMicros := parseFloat(fields[i+2])
		command := fields[i+3]
		if command == "" {
			continue
		}
		samples = append(samples, QuerySample{
			Query:   strings.ReplaceAll(command, " ", " "),
			Calls:   1,
			MeanMs:  durationMicros / 1000,
			TotalMs: durationMicros / 1000,
		})
	}
	return samples
}

func parseFloat(value string) float64 {
	parsed, _ := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return parsed
}

// parseRedisKeys extracts the key count from a "db0:keys=1,expires=0,..." value.
func parseRedisKeys(value string) float64 {
	for _, part := range strings.Split(value, ",") {
		if strings.HasPrefix(part, "keys=") {
			return parseFloat(strings.TrimPrefix(part, "keys="))
		}
	}
	return 0
}
