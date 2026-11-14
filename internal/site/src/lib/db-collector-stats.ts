import { atom } from "nanostores"
import { useEffect } from "react"
import { pb } from "@/lib/api"
import { normalizeHost } from "@/lib/database-monitors"

export type CollectorStat = {
	slow: number
	locks: number
	tps: number
	qps: number
	updated: string
}

/** Latest collector sample keyed by "host:port" so table cells can look it up. */
export const $dbCollectorStats = atom<Record<string, CollectorStat>>({})

type InventoryRow = { id: string; host: string; port: number }

/** Loads the newest db_metrics sample per collector target into the store. */
export function useDbCollectorStats() {
	useEffect(() => {
		let cancelled = false
		async function load() {
			try {
				const [inventory, metrics] = await Promise.all([
					pb.collection("db_inventory").getFullList<InventoryRow>({ sort: "name" }),
					pb.collection("db_metrics").getFullList<{
						inventory: string
						collected: string
						tps: number
						qps: number
						slow_queries: number
						blocked_sessions: number
						waiting_locks: number
					}>({ sort: "-collected" }),
				])
				const latest: Record<string, CollectorStat> = {}
				const seen = new Set<string>()
				for (const metric of metrics) {
					if (seen.has(metric.inventory)) continue
					seen.add(metric.inventory)
					const target = inventory.find((item) => item.id === metric.inventory)
					if (!target) continue
					latest[`${normalizeHost(target.host)}:${target.port}`] = {
						slow: metric.slow_queries ?? 0,
						locks: (metric.blocked_sessions ?? 0) + (metric.waiting_locks ?? 0),
						tps: metric.tps ?? 0,
						qps: metric.qps ?? 0,
						updated: metric.collected,
					}
				}
				if (!cancelled) $dbCollectorStats.set(latest)
			} catch {
				// collection absent or unauthenticated: leave the store empty
			}
		}
		load()
		const timer = setInterval(load, 30000)
		return () => {
			cancelled = true
			clearInterval(timer)
		}
	}, [])
}
