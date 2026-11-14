import { ActivityIcon, GaugeIcon, ShieldAlertIcon, ZapIcon } from "lucide-react"
import { useEffect, useState } from "react"
import { Badge } from "@/components/ui/badge"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import AddDatabaseTarget from "@/components/add-database-target"
import { pb } from "@/lib/api"
import { ENGINE_LABELS, PORT_ENGINES, formatResponse, matchInventory } from "@/lib/database-monitors"
import { cn } from "@/lib/utils"
import type { NetworkMonitorRecord } from "@/types"

type Inventory = {
	id: string
	name: string
	engine: string
	host: string
	port: number
	version: string
}

type Metric = {
	id: string
	collected: string
	tps: number
	qps: number
	slow_queries: number
	long_queries: number
	blocked_sessions: number
	waiting_locks: number
	idle_in_tx: number
	cache_hit_ratio: number
	connection_usage: number
	connections_total: number
	connections_max: number
	deadlocks_rate: number
	wal_bytes_rate: number
	temp_bytes_rate: number
	replication_lag_sec: number
	oldest_tx_sec: number
	lock_wait_sec: number
}

type Sample = {
	id: string
	collected: string
	query: string
	calls: number
	mean_ms: number
	total_ms: number
}

/**
 * Engine-level detail for the database behind a network monitor. Renders nothing
 * when no collector target matches the monitor's host and port.
 */
export default function DatabaseDetailSection({ monitor }: { monitor: NetworkMonitorRecord }) {
	const [inventory, setInventory] = useState<Inventory[] | null>(null)
	const [series, setSeries] = useState<Metric[]>([])
	const [samples, setSamples] = useState<Sample[]>([])
	const [reloadKey, setReloadKey] = useState(0)

	useEffect(() => {
		let cancelled = false
		pb.collection("db_inventory")
			.getFullList<Inventory>({ sort: "name" })
			.then((items) => {
				if (!cancelled) setInventory(items)
			})
			.catch(() => {
				if (!cancelled) setInventory([])
			})
		return () => {
			cancelled = true
		}
	}, [reloadKey])

	const matched = inventory ? matchInventory(monitor, inventory) : undefined
	const matchedId = matched?.id

	useEffect(() => {
		if (!matchedId) return
		let cancelled = false
		async function load() {
			const since = new Date(Date.now() - 3600 * 1000).toISOString()
			try {
				const [points, querySamples] = await Promise.all([
					pb.collection("db_metrics").getFullList<Metric>({
						filter: `inventory="${matchedId}" && collected>="${since}"`,
						sort: "collected",
					}),
					pb.collection("db_query_samples").getFullList<Sample>({
						filter: `inventory="${matchedId}"`,
						sort: "-collected",
					}),
				])
				if (cancelled) return
				setSeries(points)
				const newest = querySamples[0]?.collected
				setSamples(newest ? querySamples.filter((s) => s.collected === newest).slice(0, 5) : [])
			} catch {
				if (!cancelled) {
					setSeries([])
					setSamples([])
				}
			}
		}
		load()
		const timer = setInterval(load, 30000)
		return () => {
			cancelled = true
			clearInterval(timer)
		}
	}, [matchedId])

	if (!matched) {
		const engineKey = PORT_ENGINES[monitor.port] ?? ""
		return (
			<div className="grid gap-3 border-t pt-4">
				<div className="flex flex-wrap items-center gap-2">
					<GaugeIcon className="size-4" />
					<span className="text-sm font-medium">Database detail</span>
					<Badge variant="secondary" className="text-[10px] uppercase">
						{ENGINE_LABELS[engineKey] ?? (engineKey || "unknown")}
					</Badge>
					<span className="text-xs text-muted-foreground">monitor only — no collector target</span>
				</div>
				<div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
					<Tile icon={<ZapIcon className="size-3.5" />} label="Response" value={formatResponse(monitor.res)} detail="latest probe" />
					<Tile icon={<ActivityIcon className="size-3.5" />} label="Avg 1h" value={formatResponse(monitor.resAvg1h)} detail="response average" />
					<Tile icon={<GaugeIcon className="size-3.5" />} label="Max 1h" value={formatResponse(monitor.resMax1h)} detail="response maximum" />
					<Tile
						icon={<ShieldAlertIcon className="size-3.5" />}
						label="Loss 1h"
						value={`${(monitor.loss1h ?? 0).toFixed(2)}%`}
						detail="packet loss"
					/>
				</div>
				<div className="flex flex-wrap items-center justify-between gap-2">
					<p className="text-xs text-muted-foreground">
						Add a collector target with host {monitor.target} and port {monitor.port} to collect TPS, slow queries,
						locking, cache hit, and APM.
					</p>
					<AddDatabaseTarget
						preset={{
							name: `${monitor.target}-${monitor.port}`,
							host: monitor.target,
							port: monitor.port,
							engine: engineKey,
						}}
						onSaved={() => setReloadKey((key) => key + 1)}
					/>
				</div>
			</div>
		)
	}

	const engine = matched.engine || PORT_ENGINES[monitor.port] || ""
	const label = ENGINE_LABELS[engine] ?? engine
	const latest = series.length ? series[series.length - 1] : undefined
	const throughputLabel = engine === "redis" ? "Ops/s" : engine === "sqlserver" ? "Batches/s" : "TPS / QPS"
	const slowLabel = engine === "redis" ? "Slowlog entries" : "Slow queries"

	return (
		<div className="grid gap-3 border-t pt-4">
			<div className="flex flex-wrap items-center gap-2">
				<GaugeIcon className="size-4" />
				<span className="text-sm font-medium">Database detail</span>
				<Badge variant="secondary" className="text-[10px] uppercase">
					{label}
				</Badge>
				<span className="text-xs text-muted-foreground">
					{matched.name} {matched.version}
				</span>
			</div>

			<div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
				<Tile
					icon={<ZapIcon className="size-3.5" />}
					label={throughputLabel}
					value={latest ? (engine === "redis" ? latest.qps : latest.tps).toFixed(1) : "-"}
					detail={latest ? `${latest.qps.toFixed(1)} q/s · ${latest.tps.toFixed(1)} tx/s` : "waiting for sample"}
				/>
				<Tile
					icon={<ActivityIcon className="size-3.5" />}
					label={slowLabel}
					value={latest ? String(latest.slow_queries) : "-"}
					detail={latest ? `${latest.long_queries} over 5 min` : "-"}
					alert={!!latest && latest.slow_queries > 0}
				/>
				<Tile
					icon={<ShieldAlertIcon className="size-3.5" />}
					label="Blocked / locks"
					value={latest ? `${latest.blocked_sessions} / ${latest.waiting_locks}` : "-"}
					detail={latest ? `longest wait ${latest.lock_wait_sec.toFixed(0)}s` : "-"}
					alert={!!latest && latest.blocked_sessions > 0}
				/>
				<Tile
					icon={<GaugeIcon className="size-3.5" />}
					label={engine === "redis" ? "Keyspace hit" : "Cache hit"}
					value={latest ? `${latest.cache_hit_ratio.toFixed(1)}%` : "-"}
					detail={latest ? `${latest.connection_usage.toFixed(0)}% conn usage` : "-"}
				/>
			</div>

			<div className="grid gap-4 lg:grid-cols-2">
				<Spark title={`${throughputLabel} (${engine})`} points={series.map((p) => (engine === "redis" ? p.qps : p.tps))} />
				<Spark title={`${slowLabel} (active over 1 min)`} points={series.map((p) => p.slow_queries)} />
				<Spark title="Locks (blocked + waiting)" points={series.map((p) => p.blocked_sessions + p.waiting_locks)} />
				<Spark title="Cache hit ratio (%)" points={series.map((p) => p.cache_hit_ratio)} />
			</div>

			<div className="grid gap-2 text-xs text-muted-foreground sm:grid-cols-2 lg:grid-cols-4">
				<span>Idle in transaction: {latest ? latest.idle_in_tx : "-"}</span>
				<span>Deadlocks/s: {latest ? latest.deadlocks_rate.toFixed(3) : "-"}</span>
				<span>WAL bytes/s: {latest ? Math.round(latest.wal_bytes_rate) : "-"}</span>
				<span>Temp bytes/s: {latest ? Math.round(latest.temp_bytes_rate) : "-"}</span>
				<span>Oldest transaction: {latest ? `${latest.oldest_tx_sec.toFixed(0)}s` : "-"}</span>
				<span>Replication lag: {latest ? `${latest.replication_lag_sec.toFixed(0)}s` : "-"}</span>
				<span>Connections: {latest ? `${latest.connections_total}/${latest.connections_max}` : "-"}</span>
				<span>Samples: {series.length}</span>
			</div>

			<div>
				<h3 className="mb-2 text-sm font-medium">
					Top statements / slow operations ({engine})
					{samples.length === 0 && " — none recorded in the last hour"}
				</h3>
				<div className="overflow-x-auto">
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>Query</TableHead>
								<TableHead>Calls</TableHead>
								<TableHead>Mean</TableHead>
								<TableHead>Total</TableHead>
							</TableRow>
						</TableHeader>
						<TableBody>
							{samples.length === 0 && (
								<TableRow>
									<TableCell colSpan={4} className="text-center text-xs text-muted-foreground">
										No samples. Enable pg_stat_statements (PostgreSQL) or wait for a slow query.
									</TableCell>
								</TableRow>
							)}
							{samples.map((sample) => (
								<TableRow key={sample.id}>
									<TableCell className="max-w-[28rem] truncate font-mono text-xs" title={sample.query}>
										{sample.query}
									</TableCell>
									<TableCell className="text-xs">{sample.calls}</TableCell>
									<TableCell className="text-xs">{sample.mean_ms.toFixed(2)} ms</TableCell>
									<TableCell className="text-xs">{sample.total_ms.toFixed(0)} ms</TableCell>
								</TableRow>
							))}
						</TableBody>
					</Table>
				</div>
			</div>
		</div>
	)
}

function Tile({
	icon,
	label,
	value,
	detail,
	alert,
}: {
	icon: React.ReactNode
	label: string
	value: string
	detail: string
	alert?: boolean
}) {
	return (
		<div className={cn("rounded-md border p-3", alert && "border-red-500/50 bg-red-500/5")}>
			<div className="flex items-center gap-1.5 text-xs text-muted-foreground">
				{icon}
				{label}
			</div>
			<div className="mt-1 text-xl font-semibold">{value}</div>
			<p className="text-xs text-muted-foreground">{detail}</p>
		</div>
	)
}

function Spark({ title, points }: { title: string; points: number[] }) {
	const width = 300
	const height = 56
	const max = Math.max(1, ...points)
	const step = points.length > 1 ? width / (points.length - 1) : width
	const path = points
		.map((value, index) => `${index === 0 ? "M" : "L"}${(index * step).toFixed(1)},${(height - (value / max) * (height - 4) - 2).toFixed(1)}`)
		.join(" ")
	const last = points.length ? points[points.length - 1] : undefined
	return (
		<div className="rounded-md border p-3">
			<div className="flex items-center justify-between text-xs text-muted-foreground">
				<span>{title}</span>
				<span className="font-mono">{last === undefined ? "-" : last.toFixed(1)}</span>
			</div>
			<svg viewBox={`0 0 ${width} ${height}`} className="mt-2 h-14 w-full" preserveAspectRatio="none">
				<path d={path} fill="none" stroke="currentColor" strokeWidth="1.5" className="text-primary" />
			</svg>
		</div>
	)
}
