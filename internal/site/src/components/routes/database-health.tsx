import { getPagePath } from "@nanostores/router"
import { ActivityIcon, DatabaseIcon, ExternalLinkIcon, GaugeIcon, ShieldAlertIcon, ZapIcon } from "lucide-react"
import { useEffect, useMemo, useState } from "react"
import { FooterRepoLink } from "@/components/footer-repo-link"
import NetworkMonitorsTableNew from "@/components/network-monitors-table/network-monitors-table"
import { Link, $router } from "@/components/router"
import { Badge } from "@/components/ui/badge"
import { buttonVariants } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { pb } from "@/lib/api"
import { isDatabaseMonitor } from "@/lib/database-monitors"
import { cn } from "@/lib/utils"
import { useNetworkMonitors } from "@/lib/use-network-monitors"

type DBObsInventory = {
	id: string
	name: string
	environment: string
	engine: string
	host: string
	port: number
	version: string
	last_error: string
	enabled: boolean
}

type DBMetricPoint = {
	id: string
	inventory: string
	collected: string
	up: boolean
	tps: number
	qps: number
	slow_queries: number
	blocked_sessions: number
	waiting_locks: number
	idle_in_tx: number
	cache_hit_ratio: number
	connections_total: number
	connections_max: number
	connection_usage: number
	deadlocks_rate: number
	wal_bytes_rate: number
	temp_bytes_rate: number
	replication_lag_sec: number
	db_size_bytes: number
	long_queries: number
	oldest_tx_sec: number
	lock_wait_sec: number
}

type DBQuerySample = {
	id: string
	inventory: string
	collected: string
	query: string
	calls: number
	mean_ms: number
	total_ms: number
}

type DBAlert = {
	id: string
	inventory: string
	metric: string
	severity: string
	status: string
	value: number
	triggered: string
}

const WINDOWS = [
	{ label: "1 hour", hours: 1 },
	{ label: "6 hours", hours: 6 },
	{ label: "24 hours", hours: 24 },
]

export default function DatabaseHealth() {
	const { monitors, isLoading } = useNetworkMonitors({})
	const [inventory, setInventory] = useState<DBObsInventory[]>([])
	const [alerts, setAlerts] = useState<DBAlert[]>([])
	const [selectedId, setSelectedId] = useState("")
	const [windowHours, setWindowHours] = useState(1)
	const [series, setSeries] = useState<DBMetricPoint[]>([])
	const [samples, setSamples] = useState<DBQuerySample[]>([])
	const [obsError, setObsError] = useState("")

	const databaseMonitors = monitors.filter(isDatabaseMonitor)
	const responsive = databaseMonitors.filter((monitor) => (monitor.res ?? 0) > 0)
	const downCount = databaseMonitors.length - responsive.length
	const criticalAlerts = useMemo(
		() => alerts.filter((alert) => alert.severity === "critical" && alert.status !== "acknowledged"),
		[alerts]
	)

	useEffect(() => {
		document.title = "Database Health / Moonlight apps"
	}, [])

	useEffect(() => {
		let cancelled = false
		async function load() {
			try {
				const [inv, open] = await Promise.all([
					pb.collection("db_inventory").getFullList<DBObsInventory>({ sort: "name" }),
					pb.collection("db_alerts").getFullList<DBAlert>({ sort: "-triggered" }),
				])
				if (cancelled) return
				setInventory(inv)
				setAlerts(open.filter((alert) => alert.status !== "resolved"))
				setObsError("")
			} catch (error) {
				if (!cancelled) setObsError((error as Error)?.message ?? "Observability data unavailable")
			}
		}
		load()
		const timer = setInterval(load, 30000)
		return () => {
			cancelled = true
			clearInterval(timer)
		}
	}, [])

	useEffect(() => {
		if (!selectedId && inventory.length > 0) {
			setSelectedId(inventory[0].id)
		}
	}, [inventory, selectedId])

	useEffect(() => {
		if (!selectedId) return
		let cancelled = false
		async function loadSeries() {
			const since = new Date(Date.now() - windowHours * 3600 * 1000).toISOString()
			try {
				const [points, querySamples] = await Promise.all([
					pb.collection("db_metrics").getFullList<DBMetricPoint>({
						filter: `inventory="${selectedId}" && collected>="${since}"`,
						sort: "collected",
					}),
					pb.collection("db_query_samples").getFullList<DBQuerySample>({
						filter: `inventory="${selectedId}"`,
						sort: "-collected",
					}),
				])
				if (cancelled) return
				setSeries(points)
				const newest = querySamples[0]?.collected
				setSamples(newest ? querySamples.filter((s) => s.collected === newest).slice(0, 5) : [])
			} catch (error) {
				if (!cancelled) setSeries([])
			}
		}
		loadSeries()
		const timer = setInterval(loadSeries, 30000)
		return () => {
			cancelled = true
			clearInterval(timer)
		}
	}, [selectedId, windowHours])

	const latest = series.length > 0 ? series[series.length - 1] : undefined
	const selected = inventory.find((item) => item.id === selectedId)

	return (
		<div className="flex flex-col gap-4 pb-4">
			<div className="flex flex-wrap items-center justify-between gap-3">
				<div>
					<h1 className="text-xl font-semibold">Database Health</h1>
					<p className="text-sm text-muted-foreground">
						Monitors from the Databases page, plus deep PostgreSQL performance, locking, and APM.
					</p>
				</div>
				<Link
					href={getPagePath($router, "databases")}
					className={cn(buttonVariants({ variant: "outline", size: "sm" }), "gap-1.5")}
				>
					<ExternalLinkIcon className="size-4" /> Open Databases
				</Link>
			</div>

			<div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
				<SummaryCard title="Database monitors" value={String(databaseMonitors.length)} detail="Synced from /databases" />
				<SummaryCard
					title="Responsive"
					value={String(responsive.length)}
					detail={downCount > 0 ? `${downCount} down` : "All responding"}
					alert={downCount > 0}
				/>
				<SummaryCard
					title="Open alerts"
					value={String(alerts.length)}
					detail={`${criticalAlerts.length} critical`}
					alert={criticalAlerts.length > 0}
				/>
				<SummaryCard title="Inventory" value={String(inventory.length)} detail="Observability targets" />
			</div>

			<Card>
				<CardHeader className="flex-row flex-wrap items-center justify-between gap-2">
					<CardTitle className="flex items-center gap-2 text-base">
						<GaugeIcon className="size-4" /> Performance detail
						{selected && (
							<span className="text-xs font-normal text-muted-foreground">
								{selected.name} · {selected.host}:{selected.port}
							</span>
						)}
					</CardTitle>
					<div className="flex items-center gap-2">
						<Select value={selectedId} onValueChange={setSelectedId}>
							<SelectTrigger className="h-8 w-56">
								<SelectValue placeholder="Select database" />
							</SelectTrigger>
							<SelectContent>
								{inventory.map((item) => (
									<SelectItem key={item.id} value={item.id}>
										{item.name} ({item.port})
									</SelectItem>
								))}
							</SelectContent>
						</Select>
						<Select
							value={String(windowHours)}
							onValueChange={(value) => setWindowHours(Number(value))}
						>
							<SelectTrigger className="h-8 w-28">
								<SelectValue />
							</SelectTrigger>
							<SelectContent>
								{WINDOWS.map((option) => (
									<SelectItem key={option.hours} value={String(option.hours)}>
										{option.label}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
					</div>
				</CardHeader>
				<CardContent className="flex flex-col gap-4">
					{!selectedId ? (
						<p className="text-sm text-muted-foreground">
							{obsError || "No observability target yet. Register one in db_inventory to collect PostgreSQL detail."}
						</p>
					) : (
						<>
							<div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
								<MetricTile
									icon={<ZapIcon className="size-3.5" />}
									label="TPS"
									value={latest ? latest.tps.toFixed(1) : "-"}
									detail={latest ? `${latest.qps.toFixed(1)} queries/s` : "waiting for sample"}
								/>
								<MetricTile
									icon={<ActivityIcon className="size-3.5" />}
									label="Slow queries"
									value={latest ? String(latest.slow_queries) : "-"}
									detail={latest ? `${latest.long_queries} over 5 min` : "-"}
									alert={!!latest && latest.slow_queries > 0}
								/>
								<MetricTile
									icon={<ShieldAlertIcon className="size-3.5" />}
									label="Blocked / locks"
									value={latest ? `${latest.blocked_sessions} / ${latest.waiting_locks}` : "-"}
									detail={latest ? `longest wait ${latest.lock_wait_sec.toFixed(0)}s` : "-"}
									alert={!!latest && latest.blocked_sessions > 0}
								/>
								<MetricTile
									icon={<DatabaseIcon className="size-3.5" />}
									label="Cache hit"
									value={latest ? `${latest.cache_hit_ratio.toFixed(1)}%` : "-"}
									detail={latest ? `${latest.connection_usage.toFixed(0)}% conn usage` : "-"}
								/>
							</div>

							<div className="grid gap-4 lg:grid-cols-2">
								<SparklineCard
									title="TPS / QPS"
									points={series.map((point) => point.tps)}
									secondaryPoints={series.map((point) => point.qps)}
									format={(value) => value.toFixed(1)}
								/>
								<SparklineCard
									title="Slow queries (>1 min active)"
									points={series.map((point) => point.slow_queries)}
									format={(value) => String(Math.round(value))}
								/>
								<SparklineCard
									title="Locks (blocked + waiting)"
									points={series.map((point) => point.blocked_sessions + point.waiting_locks)}
									format={(value) => String(Math.round(value))}
								/>
								<SparklineCard
									title="Cache hit ratio (%)"
									points={series.map((point) => point.cache_hit_ratio)}
									format={(value) => `${value.toFixed(1)}%`}
								/>
							</div>

							<div className="grid gap-2 text-xs text-muted-foreground sm:grid-cols-2 lg:grid-cols-4">
								<span>Idle in transaction: {latest ? latest.idle_in_tx : "-"}</span>
								<span>Deadlocks/s: {latest ? latest.deadlocks_rate.toFixed(3) : "-"}</span>
								<span>WAL bytes/s: {latest ? Math.round(latest.wal_bytes_rate) : "-"}</span>
								<span>Temp bytes/s: {latest ? Math.round(latest.temp_bytes_rate) : "-"}</span>
								<span>Oldest transaction: {latest ? `${latest.oldest_tx_sec.toFixed(0)}s` : "-"}</span>
								<span>Replication lag: {latest ? `${latest.replication_lag_sec.toFixed(0)}s` : "-"}</span>
								<span>
									Connections: {latest ? `${latest.connections_total}/${latest.connections_max}` : "-"}
								</span>
								<span>Samples in window: {series.length}</span>
							</div>

							<div>
								<h3 className="mb-2 text-sm font-medium">
									Top statements (APM)
									{samples.length === 0 && " — no slow statements observed and pg_stat_statements is not installed"}
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
														No statement samples collected.
													</TableCell>
												</TableRow>
											)}
											{samples.map((sample) => (
												<TableRow key={sample.id}>
													<TableCell className="max-w-[36rem] truncate font-mono text-xs" title={sample.query}>
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
						</>
					)}
				</CardContent>
			</Card>

			{/* Single source of truth: the exact table used by the Databases page. */}
			<NetworkMonitorsTableNew
				monitors={databaseMonitors}
				isLoading={isLoading}
				title="Databases"
				emptyMessage="No databases found."
			/>

			<Card>
				<CardHeader>
					<CardTitle className="flex items-center gap-2 text-base">
						<DatabaseIcon className="size-4" /> Observability inventory
					</CardTitle>
				</CardHeader>
				<CardContent className="overflow-x-auto">
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>Database</TableHead>
								<TableHead>Environment</TableHead>
								<TableHead>Engine</TableHead>
								<TableHead>Target</TableHead>
								<TableHead>Version</TableHead>
								<TableHead>Last error</TableHead>
							</TableRow>
						</TableHeader>
						<TableBody>
							{inventory.length === 0 && (
								<TableRow>
									<TableCell colSpan={6} className="text-center text-sm text-muted-foreground">
										{obsError || "No observability targets. Add one from the API or seed db_inventory."}
									</TableCell>
								</TableRow>
							)}
							{inventory.map((item) => (
								<TableRow key={item.id}>
									<TableCell className="font-medium">{item.name}</TableCell>
									<TableCell className="text-xs">{item.environment || "-"}</TableCell>
									<TableCell className="text-xs">{item.engine || "-"}</TableCell>
									<TableCell className="text-xs">
										{item.host}:{item.port}
									</TableCell>
									<TableCell className="text-xs">{item.version || "-"}</TableCell>
									<TableCell className="max-w-64 truncate text-xs text-muted-foreground">
										{item.last_error || "-"}
									</TableCell>
								</TableRow>
							))}
						</TableBody>
					</Table>
				</CardContent>
			</Card>

			<Card>
				<CardHeader>
					<CardTitle className="flex items-center gap-2 text-base">
						<ShieldAlertIcon className="size-4" /> Alerts
					</CardTitle>
				</CardHeader>
				<CardContent className="overflow-x-auto">
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>Database</TableHead>
								<TableHead>Metric</TableHead>
								<TableHead>Severity</TableHead>
								<TableHead>Value</TableHead>
								<TableHead>Status</TableHead>
								<TableHead>Triggered</TableHead>
							</TableRow>
						</TableHeader>
						<TableBody>
							{alerts.length === 0 && (
								<TableRow>
									<TableCell colSpan={6} className="text-center text-sm text-muted-foreground">
										No open database alerts.
									</TableCell>
								</TableRow>
							)}
							{alerts.map((alert) => (
								<TableRow key={alert.id}>
									<TableCell className="text-sm">
										{inventory.find((item) => item.id === alert.inventory)?.name ?? alert.inventory}
									</TableCell>
									<TableCell className="text-xs">{alert.metric}</TableCell>
									<TableCell>
										<Badge variant="secondary">{alert.severity}</Badge>
									</TableCell>
									<TableCell className="text-xs">{alert.value?.toFixed(2)}</TableCell>
									<TableCell>
										<Badge variant="secondary">{alert.status}</Badge>
									</TableCell>
									<TableCell className="text-xs text-muted-foreground">
										{new Date(alert.triggered).toLocaleString(undefined, { timeZone: "Asia/Jakarta" })}
									</TableCell>
								</TableRow>
							))}
						</TableBody>
					</Table>
				</CardContent>
			</Card>

			<FooterRepoLink />
		</div>
	)
}

function SummaryCard({
	title,
	value,
	detail,
	alert,
}: {
	title: string
	value: string
	detail: string
	alert?: boolean
}) {
	return (
		<Card className={alert ? "border-red-500/50" : undefined}>
			<CardHeader className="pb-2">
				<CardTitle className="text-xs font-medium text-muted-foreground">{title}</CardTitle>
			</CardHeader>
			<CardContent>
				<div className="text-2xl font-semibold">{value}</div>
				<p className="text-xs text-muted-foreground">{detail}</p>
			</CardContent>
		</Card>
	)
}

function MetricTile({
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

/** Dependency-free sparkline so the page works without a chart library. */
function SparklineCard({
	title,
	points,
	secondaryPoints,
	format,
}: {
	title: string
	points: number[]
	secondaryPoints?: number[]
	format: (value: number) => string
}) {
	const width = 320
	const height = 64
	const path = buildPath(points, width, height)
	const secondaryPath = secondaryPoints ? buildPath(secondaryPoints, width, height, points) : ""
	const max = points.length ? Math.max(...points) : 0
	const last = points.length ? points[points.length - 1] : undefined

	return (
		<div className="rounded-md border p-3">
			<div className="flex items-center justify-between text-xs text-muted-foreground">
				<span>{title}</span>
				<span className="font-mono">{last === undefined ? "-" : format(last)}</span>
			</div>
			<svg viewBox={`0 0 ${width} ${height}`} className="mt-2 h-16 w-full" preserveAspectRatio="none">
				<path d={path} fill="none" stroke="currentColor" strokeWidth="1.5" className="text-primary" />
				{secondaryPath && (
					<path d={secondaryPath} fill="none" stroke="currentColor" strokeWidth="1" className="text-muted-foreground" />
				)}
			</svg>
			<div className="text-[10px] text-muted-foreground">
				peak {format(max)}
			</div>
		</div>
	)
}

function buildPath(points: number[], width: number, height: number, scaleWith?: number[]): string {
	if (points.length === 0) return ""
	const max = Math.max(1, ...points, ...(scaleWith ?? []))
	const step = points.length > 1 ? width / (points.length - 1) : width
	return points
		.map((value, index) => {
			const x = index * step
			const y = height - (value / max) * (height - 4) - 2
			return `${index === 0 ? "M" : "L"}${x.toFixed(1)},${y.toFixed(1)}`
		})
		.join(" ")
}
