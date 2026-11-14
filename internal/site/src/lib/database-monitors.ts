import type { NetworkMonitorRecord } from "@/types"

/** Ports that identify a database service in the network monitor list. */
export const DATABASE_PORTS = new Set([5432, 3306, 6379, 1433, 1521, 50000, 3000, 5006, 5008, 5009, 27017, 9200])

/** Telnet monitors are always database/service monitors, whatever port they use. */
export const isDatabaseMonitor = (monitor: NetworkMonitorRecord) =>
	monitor.protocol === "telnet" || DATABASE_PORTS.has(monitor.port)
