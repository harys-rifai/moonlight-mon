package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pocketbase/dbx"
)

func mustJSON(v any) typesJSON {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return typesJSON(b)
}

type typesJSON json.RawMessage

func (j typesJSON) Value() (interface{}, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return string(j), nil
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("POSTGRES_URL")
	}
	if dsn == "" {
		dsn = "postgres://postgres:Password09@localhost:5006/server?sslmode=disable"
	}
	db, err := dbx.Open("pgx", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	now := time.Now().UTC()
	ts := now.Format("2006-01-02 15:04:05.000Z")
	ago1h := now.Add(-1 * time.Hour).Format("2006-01-02 15:04:05.000Z")
	ago30m := now.Add(-30 * time.Minute).Format("2006-01-02 15:04:05.000Z")
	epoch := now.Unix()

	// find existing user for relations
	var users []struct {
		ID    string `db:"id"`
		Email string `db:"email"`
	}
	if err := db.NewQuery("select id, email from users order by created asc limit 5").All(&users); err != nil {
		log.Fatalf("query users: %v", err)
	}
	if len(users) == 0 {
		log.Fatal("no users found; create/login once or seed a user first so user-linked tables can be populated")
	}
	userID := users[0].ID
	fmt.Printf("using user: %s (%s)\n", userID, users[0].Email)

	// helper exec
	exec := func(q string, params dbx.Params) {
		if _, err := db.NewQuery(q).Bind(params).Execute(); err != nil {
			log.Fatalf("exec failed: %v\nquery: %s\nparams: %#v", err, q, params)
		}
	}

	// cleanup sample records by fixed IDs/prefixes to be idempotent
	cleanups := []string{
		`delete from network_monitor_stats where id like 'seednmstat%' or monitor in ('seedmonpg01','seedmonmy01','seedmonrd01','seedmonms01')`,
		`delete from network_monitors where id in ('seedmonpg01','seedmonmy01','seedmonrd01','seedmonms01')`,
		`delete from container_stats where id like 'seedcstat%'`,
		`delete from system_stats where id like 'seedsstat%'`,
		`delete from containers where id like 'seedctr%'`,
		`delete from alerts_history where id like 'seedahist%'`,
		`delete from alerts where id like 'seedalert%'`,
		`delete from quiet_hours where id like 'seedquiet%'`,
		`delete from smart_devices where id like 'seedsmart%'`,
		`delete from zfs_pools where id like 'seedzpool%'`,
		`delete from systemd_services where id like 'seedsvc%'`,
		`delete from system_details where id like 'seedsysdt%'`,
		`delete from fingerprints where id like 'seedfngp%'`,
		`delete from universal_tokens where id like 'seedutok%'`,
		`delete from user_settings where id like 'seeduset%' and "user"={:uid}`,
		`delete from systems where id in ('seedsysdb001','seedsysapp01','seedsyscache1')`,
	}
	for _, c := range cleanups {
		exec(c, dbx.Params{"uid": userID})
	}

	sysUsers, _ := json.Marshal([]string{userID})

	// systems
	systems := []dbx.Params{
		{"id": "seedsysdb001", "name": "prod-postgres-01", "status": "up", "host": "10.20.30.11", "port": "45876", "info": mustJSON(map[string]any{"h": "prod-postgres-01", "k": "6.8.0-60-generic", "c": 8, "t": 16, "m": "Intel Xeon Silver 4314", "u": 864000, "cpu": 18.4, "mp": 62.1, "dp": 48.7, "os": 0, "v": "0.12.0"}), "users": string(sysUsers), "created": ts, "updated": ts},
		{"id": "seedsysapp01", "name": "prod-app-01", "status": "up", "host": "10.20.30.21", "port": "45876", "info": mustJSON(map[string]any{"h": "prod-app-01", "k": "6.8.0-60-generic", "c": 4, "t": 8, "m": "AMD EPYC 7763", "u": 432000, "cpu": 34.2, "mp": 71.5, "dp": 39.2, "os": 0, "v": "0.12.0"}), "users": string(sysUsers), "created": ts, "updated": ts},
		{"id": "seedsyscache1", "name": "prod-redis-cache-01", "status": "up", "host": "10.20.30.31", "port": "45876", "info": mustJSON(map[string]any{"h": "prod-redis-cache-01", "k": "6.8.0-60-generic", "c": 4, "t": 8, "m": "Intel Xeon E5-2686 v4", "u": 259200, "cpu": 12.7, "mp": 54.0, "dp": 22.4, "os": 0, "v": "0.12.0"}), "users": string(sysUsers), "created": ts, "updated": ts},
	}
	for _, p := range systems {
		exec(`insert into systems (id,name,status,host,port,info,users,created,updated) values ({:id},{:name},{:status},{:host},{:port},{:info}::jsonb,{:users}::jsonb,{:created}::timestamp,{:updated}::timestamp)`, p)
	}

	// system_details
	details := []dbx.Params{
		{"id": "seedsysdt001", "system": "seedsysdb001", "hostname": "prod-postgres-01", "kernel": "6.8.0-60-generic", "cores": 8, "threads": 16, "cpu": "Intel Xeon Silver 4314", "os": 0, "os_name": "Ubuntu 24.04 LTS", "arch": "x86_64", "memory": 34359738368, "podman": false, "updated": ts, "sockets": 1},
		{"id": "seedsysdt002", "system": "seedsysapp01", "hostname": "prod-app-01", "kernel": "6.8.0-60-generic", "cores": 4, "threads": 8, "cpu": "AMD EPYC 7763", "os": 0, "os_name": "Ubuntu 24.04 LTS", "arch": "x86_64", "memory": 17179869184, "podman": false, "updated": ts, "sockets": 1},
		{"id": "seedsysdt003", "system": "seedsyscache1", "hostname": "prod-redis-cache-01", "kernel": "6.8.0-60-generic", "cores": 4, "threads": 8, "cpu": "Intel Xeon E5-2686 v4", "os": 0, "os_name": "Debian GNU/Linux 12", "arch": "x86_64", "memory": 17179869184, "podman": false, "updated": ts, "sockets": 1},
	}
	for _, p := range details {
		exec(`insert into system_details (id,system,hostname,kernel,cores,threads,cpu,os,os_name,arch,memory,podman,updated,sockets) values ({:id},{:system},{:hostname},{:kernel},{:cores},{:threads},{:cpu},{:os},{:os_name},{:arch},{:memory},{:podman},{:updated}::timestamp,{:sockets})`, p)
	}

	// containers (real names)
	containers := []dbx.Params{
		{"id": "seedctrpg01a", "system": "seedsysdb001", "name": "postgres-primary", "status": "running", "health": 1, "cpu": 8.4, "memory": 4294967296.0, "net": 12582912.0, "image": "postgres:16.4", "ports": "5432/tcp", "updated": epoch, "updatable": false},
		{"id": "seedctrpg01b", "system": "seedsysdb001", "name": "pgbouncer", "status": "running", "health": 1, "cpu": 1.2, "memory": 134217728.0, "net": 2097152.0, "image": "bitnami/pgbouncer:1.23.1", "ports": "6432/tcp", "updated": epoch, "updatable": false},
		{"id": "seedctrapp01", "system": "seedsysapp01", "name": "moonlight-api", "status": "running", "health": 1, "cpu": 14.6, "memory": 2147483648.0, "net": 18874368.0, "image": "ghcr.io/moonlight/api:1.8.2", "ports": "8080/tcp", "updated": epoch, "updatable": true},
		{"id": "seedctrapp02", "system": "seedsysapp01", "name": "nginx-proxy", "status": "running", "health": 1, "cpu": 2.1, "memory": 268435456.0, "net": 7340032.0, "image": "nginx:1.27-alpine", "ports": "80/tcp,443/tcp", "updated": epoch, "updatable": false},
		{"id": "seedctrcache1", "system": "seedsyscache1", "name": "redis-cache", "status": "running", "health": 1, "cpu": 3.7, "memory": 3221225472.0, "net": 9437184.0, "image": "redis:7.4-alpine", "ports": "6379/tcp", "updated": epoch, "updatable": false},
		{"id": "seedctrprom1", "system": "seedsyscache1", "name": "prometheus", "status": "running", "health": 1, "cpu": 5.3, "memory": 1073741824.0, "net": 3145728.0, "image": "prom/prometheus:v2.54.1", "ports": "9090/tcp", "updated": epoch, "updatable": true},
	}
	for _, p := range containers {
		exec(`insert into containers (id,system,name,status,health,cpu,memory,net,image,ports,updated,updatable) values ({:id},{:system},{:name},{:status},{:health},{:cpu},{:memory},{:net},{:image},{:ports},{:updated},{:updatable})`, p)
	}

	// systemd_services (actual DB columns: cpu,cpuPeak,id,memPeak,memory,name,state,sub,system,updated numeric)
	svcBase := []struct {
		id, system, name string
		cpu, cpuPeak     float64
		mem, memPeak     float64
		state, sub       float64
	}{
		{"seedsvcpg01a", "seedsysdb001", "postgresql.service", 1.8, 12.3, 2048, 4096, 1, 0},
		{"seedsvcpg01b", "seedsysdb001", "pgbouncer.service", 0.4, 2.1, 96, 180, 1, 0},
		{"seedsvcapp01", "seedsysapp01", "moonlight-api.service", 6.2, 28.5, 1536, 3072, 1, 0},
		{"seedsvcnginx1", "seedsysapp01", "nginx.service", 0.7, 4.0, 128, 256, 1, 0},
		{"seedsvcredis1", "seedsyscache1", "redis-server.service", 1.1, 9.4, 2867, 3500, 1, 0},
		{"seedsvcprom1", "seedsyscache1", "prometheus.service", 2.3, 11.0, 820, 1400, 1, 0},
	}
	for _, s := range svcBase {
		exec(`insert into systemd_services (id,system,name,state,sub,cpu,"cpuPeak",memory,"memPeak",updated) values ({:id},{:system},{:name},{:state},{:sub},{:cpu},{:cpuPeak},{:memory},{:memPeak},{:updated})`,
			dbx.Params{"id": s.id, "system": s.system, "name": s.name, "state": s.state, "sub": s.sub, "cpu": s.cpu, "cpuPeak": s.cpuPeak, "memory": s.mem, "memPeak": s.memPeak, "updated": epoch})
	}

	// smart_devices
	smart := []dbx.Params{
		{"id": "seedsmart001", "system": "seedsysdb001", "name": "/dev/nvme0n1", "model": "Samsung SSD 990 PRO 2TB", "state": "Passed", "capacity": 2000398934016.0, "temp": 38, "firmware": "4B2QJXD7", "serial": "S6Z1NF0X123456A", "type": "nvme", "hours": 1240, "cycles": 210, "attributes": mustJSON(map[string]any{"Percentage Used": 3, "Available Spare": 100, "Media Errors": 0}), "updated": ts},
		{"id": "seedsmart002", "system": "seedsysapp01", "name": "/dev/sda", "model": "Crucial MX500 1TB", "state": "Passed", "capacity": 1000204886016.0, "temp": 34, "firmware": "M3CR046", "serial": "2142E5D7ABCD", "type": "sata", "hours": 8420, "cycles": 530, "attributes": mustJSON(map[string]any{"Power_On_Hours": 8420, "Reallocated_Sector_Ct": 0, "Wear_Leveling_Count": 97}), "updated": ts},
		{"id": "seedsmart003", "system": "seedsyscache1", "name": "/dev/nvme1n1", "model": "WD Black SN850X 1TB", "state": "Passed", "capacity": 1000204886016.0, "temp": 41, "firmware": "620361WD", "serial": "23184H800456", "type": "nvme", "hours": 3120, "cycles": 180, "attributes": mustJSON(map[string]any{"Percentage Used": 5, "Available Spare": 99, "Unsafe Shutdowns": 2}), "updated": ts},
	}
	for _, p := range smart {
		exec(`insert into smart_devices (id,system,name,model,state,capacity,temp,firmware,serial,type,hours,cycles,attributes,updated) values ({:id},{:system},{:name},{:model},{:state},{:capacity},{:temp},{:firmware},{:serial},{:type},{:hours},{:cycles},{:attributes}::jsonb,{:updated}::timestamp)`, p)
	}

	// zfs_pools
	pools := []dbx.Params{
		{"id": "seedzpool001", "system": "seedsysdb001", "name": "tank", "display_name": "tank (PostgreSQL data)", "health": "ONLINE", "status": "ONLINE", "size": 4000787030016.0, "alloc": 1950384128000.0, "free": 2050402902016.0, "raw": false, "scrub": mustJSON(map[string]any{"state": "finished", "start": ago1h}), "datasets": mustJSON([]map[string]any{{"name": "tank/pgdata", "used": 1200.5, "avail": 1800.0}}), "vdevs": mustJSON([]map[string]any{{"name": "mirror-0", "state": "ONLINE"}}), "updated": ts, "details_updated": ts},
		{"id": "seedzpool002", "system": "seedsysapp01", "name": "apps", "display_name": "apps (Application storage)", "health": "ONLINE", "status": "ONLINE", "size": 2000398934016.0, "alloc": 784000000000.0, "free": 1216398934016.0, "raw": false, "scrub": mustJSON(map[string]any{"state": "finished", "start": ago1h}), "datasets": mustJSON([]map[string]any{{"name": "apps/releases", "used": 420.0, "avail": 1100.0}}), "vdevs": mustJSON([]map[string]any{{"name": "mirror-0", "state": "ONLINE"}}), "updated": ts, "details_updated": ts},
		{"id": "seedzpool003", "system": "seedsyscache1", "name": "cachepool", "display_name": "cachepool (Redis snapshots)", "health": "ONLINE", "status": "ONLINE", "size": 1000204886016.0, "alloc": 224000000000.0, "free": 776204886016.0, "raw": false, "scrub": mustJSON(map[string]any{"state": "finished", "start": ago1h}), "datasets": mustJSON([]map[string]any{{"name": "cachepool/rdb", "used": 180.0, "avail": 720.0}}), "vdevs": mustJSON([]map[string]any{{"name": "stripe", "state": "ONLINE"}}), "updated": ts, "details_updated": ts},
	}
	for _, p := range pools {
		exec(`insert into zfs_pools (id,system,name,display_name,health,status,size,alloc,free,raw,scrub,datasets,vdevs,updated,details_updated) values ({:id},{:system},{:name},{:display_name},{:health},{:status},{:size},{:alloc},{:free},{:raw},{:scrub}::jsonb,{:datasets}::jsonb,{:vdevs}::jsonb,{:updated}::timestamp,{:details_updated}::timestamp)`, p)
	}

	// fingerprints & universal_tokens
	exec(`insert into fingerprints (id,system,token,fingerprint,updated) values ({:id},{:system},{:token},{:fingerprint},{:updated}::timestamp)`,
		dbx.Params{"id": "seedfngp001", "system": "seedsysdb001", "token": "seedfp-token-db01-aaaaaaaaaa", "fingerprint": "fp:prod-postgres-01:aa11bb22", "updated": ts})
	exec(`insert into fingerprints (id,system,token,fingerprint,updated) values ({:id},{:system},{:token},{:fingerprint},{:updated}::timestamp)`,
		dbx.Params{"id": "seedfngp002", "system": "seedsysapp01", "token": "seedfp-token-app01-bbbbbbbbbb", "fingerprint": "fp:prod-app-01:cc33dd44", "updated": ts})
	exec(`insert into universal_tokens (id,user,token,created) values ({:id},{:user},{:token},{:created}::timestamp)`,
		dbx.Params{"id": "seedutok000001", "user": userID, "token": "seed-universal-token-ops-readonly", "created": ts})

	// network_monitors (real database/service names)
	monitors := []dbx.Params{
		{"id": "seedmonpg01", "system": "seedsysdb001", "target": "prod-postgres-01.internal", "protocol": "tcp", "port": 5432, "interval": 30, "enabled": true, "res": 2.4, "resAvg1h": 2.6, "resMin1h": 1.9, "resMax1h": 5.1, "loss1h": 0, "created": ts, "updated": ago30m, "certInfo": nil, "server": "PostgreSQL 16", "skipTlsVerify": false},
		{"id": "seedmonmy01", "system": "seedsysapp01", "target": "mysql-reporting.internal", "protocol": "tcp", "port": 3306, "interval": 30, "enabled": true, "res": 3.1, "resAvg1h": 3.4, "resMin1h": 2.7, "resMax1h": 6.8, "loss1h": 0, "created": ts, "updated": ago30m, "certInfo": nil, "server": "MySQL 8.0", "skipTlsVerify": false},
		{"id": "seedmonrd01", "system": "seedsyscache1", "target": "prod-redis-cache-01.internal", "protocol": "tcp", "port": 6379, "interval": 30, "enabled": true, "res": 1.2, "resAvg1h": 1.3, "resMin1h": 0.9, "resMax1h": 2.4, "loss1h": 0, "created": ts, "updated": ago30m, "certInfo": nil, "server": "Redis 7.4", "skipTlsVerify": false},
		{"id": "seedmonms01", "system": "seedsysapp01", "target": "sqlserver-warehouse.internal", "protocol": "tcp", "port": 1433, "interval": 60, "enabled": true, "res": 8.7, "resAvg1h": 9.1, "resMin1h": 7.8, "resMax1h": 14.2, "loss1h": 0.5, "created": ts, "updated": ago30m, "certInfo": nil, "server": "SQL Server 2022", "skipTlsVerify": false},
	}
	for _, p := range monitors {
		exec(`insert into network_monitors (id,system,target,protocol,port,interval,enabled,res,"resAvg1h","resMin1h","resMax1h","loss1h",created,updated,"certInfo",server,"skipTlsVerify") values ({:id},{:system},{:target},{:protocol},{:port},{:interval},{:enabled},{:res},{:resAvg1h},{:resMin1h},{:resMax1h},{:loss1h},{:created}::timestamp,{:updated}::timestamp,{:certInfo}::jsonb,{:server},{:skipTlsVerify})`, p)
	}

	// network_monitor_stats
	statsRows := []dbx.Params{
		{"id": "seednmstat01", "system": "seedsysdb001", "monitor": "seedmonpg01", "total_count": 120, "success_count": 120, "res_sum": 312000, "res_min": 1900, "res_max": 5100, "type": "1m", "created": epoch},
		{"id": "seednmstat02", "system": "seedsysapp01", "monitor": "seedmonmy01", "total_count": 120, "success_count": 120, "res_sum": 408000, "res_min": 2700, "res_max": 6800, "type": "1m", "created": epoch},
		{"id": "seednmstat03", "system": "seedsyscache1", "monitor": "seedmonrd01", "total_count": 120, "success_count": 120, "res_sum": 156000, "res_min": 900, "res_max": 2400, "type": "1m", "created": epoch},
		{"id": "seednmstat04", "system": "seedsysapp01", "monitor": "seedmonms01", "total_count": 60, "success_count": 59, "res_sum": 546000, "res_min": 7800, "res_max": 14200, "type": "1m", "created": epoch},
	}
	for _, p := range statsRows {
		exec(`insert into network_monitor_stats (id,system,monitor,total_count,success_count,res_sum,res_min,res_max,type,created) values ({:id},{:system},{:monitor},{:total_count},{:success_count},{:res_sum},{:res_min},{:res_max},{:type},{:created})`, p)
	}

	// system_stats / container_stats
	sysStatsPayload := mustJSON(map[string]any{"cpu": 21.3, "m": 16.0, "mu": 9.8, "mp": 61.2, "d": 500.0, "du": 240.0, "dp": 48.0, "ns": 1200000.0, "nr": 980000.0, "la": []float64{0.82, 0.74, 0.69}})
	for i, sid := range []string{"seedsysdb001", "seedsysapp01", "seedsyscache1"} {
		exec(`insert into system_stats (id,system,stats,type,created,updated) values ({:id},{:system},{:stats}::jsonb,{:type},{:created}::timestamp,{:updated}::timestamp)`,
			dbx.Params{"id": fmt.Sprintf("seedsstat%02d1", i+1), "system": sid, "stats": sysStatsPayload, "type": "1m", "created": ts, "updated": ts})
	}
	ctrStatsPayload := mustJSON([]map[string]any{{"n": "postgres-primary", "c": 8.4, "m": 4096.0}, {"n": "pgbouncer", "c": 1.2, "m": 128.0}})
	exec(`insert into container_stats (id,system,stats,type,created,updated) values ({:id},{:system},{:stats}::jsonb,{:type},{:created}::timestamp,{:updated}::timestamp)`,
		dbx.Params{"id": "seedcstat001", "system": "seedsysdb001", "stats": ctrStatsPayload, "type": "1m", "created": ts, "updated": ts})

	// alerts & alerts_history
	alerts := []dbx.Params{
		{"id": "seedalert001", "user": userID, "system": "seedsysdb001", "name": "CPU", "value": 85, "min": 5, "triggered": false, "state": mustJSON(map[string]any{}), "pending_since": sql.NullString{}, "created": ts, "updated": ts},
		{"id": "seedalert002", "user": userID, "system": "seedsyscache1", "name": "Memory", "value": 90, "min": 5, "triggered": false, "state": mustJSON(map[string]any{}), "pending_since": sql.NullString{}, "created": ts, "updated": ts},
		{"id": "seedalert003", "user": userID, "system": "seedsysapp01", "name": "Status", "value": 0, "min": 1, "triggered": false, "state": mustJSON(map[string]any{}), "pending_since": sql.NullString{}, "created": ts, "updated": ts},
	}
	for _, p := range alerts {
		exec(`insert into alerts (id,"user",system,name,value,min,triggered,state,pending_since,created,updated) values ({:id},{:user},{:system},{:name},{:value},{:min},{:triggered},{:state}::jsonb,{:pending_since},{:created}::timestamp,{:updated}::timestamp)`, p)
	}
	exec(`insert into alerts_history (id,"user",system,alert_id,name,monitor_name,value,resolved,created) values ({:id},{:user},{:system},{:alert_id},{:name},{:monitor_name},{:value},{:resolved},{:created}::timestamp)`,
		dbx.Params{"id": "seedahist001", "user": userID, "system": "seedsysdb001", "alert_id": "seedalert001", "name": "CPU", "monitor_name": "", "value": 91.2, "resolved": ago30m, "created": ago1h})
	exec(`insert into alerts_history (id,"user",system,alert_id,name,monitor_name,value,resolved,created) values ({:id},{:user},{:system},{:alert_id},{:name},{:monitor_name},{:value},{:resolved},{:created}::timestamp)`,
		dbx.Params{"id": "seedahist002", "user": userID, "system": "seedsysapp01", "alert_id": "seedmonms01", "name": "NetworkMonitorLoss", "monitor_name": "sqlserver-warehouse.internal", "value": 5.0, "resolved": sql.NullString{}, "created": ago30m})

	// user_settings & quiet_hours
	settings := mustJSON(map[string]any{"emails": []string{users[0].Email}, "webhooks": []any{}, "chartTime": "1h"})
	exec(`insert into user_settings (id,"user",settings,created,updated) values ({:id},{:user},{:settings}::jsonb,{:created}::timestamp,{:updated}::timestamp)`,
		dbx.Params{"id": "seeduset00001", "user": userID, "settings": settings, "created": ts, "updated": ts})
	exec(`insert into quiet_hours (id,"user",system,type,start,"end") values ({:id},{:user},{:system},{:type},{:start}::timestamp,{:end}::timestamp)`,
		dbx.Params{"id": "seedquiet001", "user": userID, "system": "seedsysdb001", "type": "daily", "start": now.Add(22 * time.Hour).Format("2006-01-02 15:04:05.000Z"), "end": now.Add(30 * time.Hour).Format("2006-01-02 15:04:05.000Z")})

	fmt.Println("sample data inserted for tables: users(reused), systems, system_details, containers, systemd_services, smart_devices, zfs_pools, fingerprints, universal_tokens, network_monitors, network_monitor_stats, system_stats, container_stats, alerts, alerts_history, user_settings, quiet_hours")
}
