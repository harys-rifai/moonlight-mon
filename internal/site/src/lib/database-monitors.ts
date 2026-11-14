import type { NetworkMonitorRecord } from "@/types"

/** Ports that identify a database service in the network monitor list. */
export const DATABASE_PORTS = new Set([5432, 3306, 6379, 1433, 1521, 50000, 3000, 5006, 5008, 5009, 27017, 9200])

/** Telnet monitors are always database/service monitors, whatever port they use. */
export const isDatabaseMonitor = (monitor: NetworkMonitorRecord) =>
	monitor.protocol === "telnet" || DATABASE_PORTS.has(monitor.port)

/** Display labels for collector engines. */
export const ENGINE_LABELS: Record<string, string> = {
	postgres: "PostgreSQL",
	mysql: "MySQL",
	mariadb: "MariaDB",
	redis: "Redis",
	sqlserver: "SQL Server",
	oracle: "Oracle",
	db2: "DB2",
	mongodb: "MongoDB",
	elasticsearch: "Elasticsearch",
	aerospike: "Aerospike",
	telnet: "Telnet",
}

/** Infer engine from the monitored port when no collector is registered. */
export const PORT_ENGINES: Record<number, string> = {
	5432: "postgres",
	5006: "postgres",
	5008: "postgres",
	5009: "postgres",
	3306: "mysql",
	6379: "redis",
	1433: "sqlserver",
	1521: "oracle",
	50000: "db2",
	27017: "mongodb",
	9200: "elasticsearch",
	3000: "aerospike",
}

export function normalizeHost(host: string) {
	const value = host.replace(/^\[|\]$/g, "").toLowerCase()
	return value === "127.0.0.1" || value === "::1" ? "localhost" : value
}

/** Find the collector target that shares a monitor's host and port. */
export function matchInventory<T extends { host: string; port: number }>(
	monitor: NetworkMonitorRecord,
	inventory: T[]
): T | undefined {
	return inventory.find(
		(item) => item.port === monitor.port && normalizeHost(item.host) === normalizeHost(monitor.target)
	)
}

export function formatResponse(value?: number) {
	if (!value) return "-"
	if (value < 1000) return `${value.toFixed(1)} µs`
	return `${(value / 1000).toFixed(1)} ms`
}
