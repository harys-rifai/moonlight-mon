import { useEffect, useMemo, useState } from "react"
import { DatabaseIcon, RefreshCwIcon, PlusIcon, ShieldAlertIcon } from "lucide-react"
import { pb, isAdmin } from "@/lib/api"
import { formatBytes } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
	Dialog,
	DialogContent,
	DialogHeader,
	DialogTitle,
	DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table"
import { useToast } from "@/components/ui/use-toast"
import { FooterRepoLink } from "@/components/footer-repo-link"

type DBInventory = {
	id: string
	name: string
	application: string
	environment: string
	engine: string
	host: string
	port: number
	dbname: string
	username: string
	password: string
	sslmode: string
	owner: string
	ha_mode: string
	cloud_provider: string
	group_name: string
	notes: string
	enabled: boolean
	version: string
	last_error: string
}

type DBMetric = {
	id: string
	inventory: string
	collected: string
	up: boolean
	connection_usage: number
	connections_total: number
	connections_max: number
	long_queries: number
	blocked_sessions: number
	replication_lag_sec: number
	db_size_bytes: number
	oldest_tx_sec: number
}

type DBAlert = {
	id: string
	inventory: string
	metric: string
	severity: string
	status: string
	value: number
	message: string
	triggered: string
}

const emptyForm = {
	name: "",
	application: "",
	environment: "DEV",
	engine: "postgres",
	host: "localhost",
	port: "5432",
	dbname: "postgres",
	username: "postgres",
	password: "",
	sslmode: "disable",
	owner: "",
	ha_mode: "",
	cloud_provider: "",
	group_name: "",
	notes: "",
	enabled: true,
}

const statusColor: Record<string, string> = {
	critical: "bg-red-500/15 text-red-600 dark:text-red-400",
	warning: "bg-amber-500/15 text-amber-600 dark:text-amber-400",
	acknowledged: "bg-blue-500/15 text-blue-600 dark:text-blue-400",
	resolved: "bg-green-500/15 text-green-700 dark:text-green-400",
}

export default function DatabaseHealth() {
	const { toast } = useToast()
	const [inventory, setInventory] = useState<DBInventory[]>([])
	const [metrics, setMetrics] = useState<DBMetric[]>([])
	const [alerts, setAlerts] = useState<DBAlert[]>([])
	const [loading, setLoading] = useState(true)
	const [open, setOpen] = useState(false)
	const [editing, setEditing] = useState<DBInventory | null>(null)
	const [form, setForm] = useState({ ...emptyForm })
	const admin = isAdmin()

	async function load() {
		try {
			const [inv, mets, al] = await Promise.all([
				pb.collection("db_inventory").getFullList<DBInventory>({ sort: "name" }),
				pb.collection("db_metrics").getFullList<DBMetric>({ sort: "-collected" }),
				pb.collection("db_alerts").getFullList<DBAlert>({ sort: "-triggered" }),
			])
			setInventory(inv)
			setMetrics(mets)
			setAlerts(al.filter((a) => a.status !== "resolved"))
		} catch (error) {
			console.error("Failed to load database observability data", error)
		} finally {
			setLoading(false)
		}
	}

	useEffect(() => {
		document.title = "Database Health / Moonlight apps"
		load()
		const timer = setInterval(load, 30000)
		return () => clearInterval(timer)
	}, [])

	const latest = useMemo(() => {
		const map = new Map<string, DBMetric>()
		for (const metric of metrics) {
			if (!map.has(metric.inventory)) map.set(metric.inventory, metric)
		}
		return map
	}, [metrics])

	const upCount = inventory.filter((item) => latest.get(item.id)?.up).length
	const criticalAlerts = alerts.filter((a) => a.severity === "critical" && a.status !== "acknowledged")

	function startAdd() {
		setEditing(null)
		setForm({ ...emptyForm })
		setOpen(true)
	}

	function startEdit(item: DBInventory) {
		setEditing(item)
		setForm({
			name: item.name ?? "",
			application: item.application ?? "",
			environment: item.environment || "DEV",
			engine: item.engine || "postgres",
			host: item.host ?? "",
			port: String(item.port || 5432),
			dbname: item.dbname ?? "",
			username: item.username ?? "",
			password: "",
			sslmode: item.sslmode || "disable",
			owner: item.owner ?? "",
			ha_mode: item.ha_mode ?? "",
			cloud_provider: item.cloud_provider ?? "",
			group_name: item.group_name ?? "",
			notes: item.notes ?? "",
			enabled: item.enabled ?? true,
		})
		setOpen(true)
	}

	async function save() {
		const payload = {
			name: form.name.trim(),
			application: form.application.trim(),
			environment: form.environment,
			engine: form.engine,
			host: form.host.trim(),
			port: Number(form.port) || 0,
			dbname: form.dbname.trim(),
			username: form.username.trim(),
			sslmode: form.sslmode,
			owner: form.owner.trim(),
			ha_mode: form.ha_mode.trim(),
			cloud_provider: form.cloud_provider.trim(),
			group_name: form.group_name.trim(),
			notes: form.notes.trim(),
			enabled: form.enabled,
		}
		if (form.password) {
			;(payload as Record<string, unknown>).password = form.password
		}
		try {
			if (editing) {
				await pb.collection("db_inventory").update(editing.id, payload)
			} else {
				await pb.collection("db_inventory").create(payload)
			}
			setOpen(false)
			await load()
		} catch (error) {
			toast({
				variant: "destructive",
				title: "Save failed",
				description: (error as Error)?.message,
			})
		}
	}

	async function remove(item: DBInventory) {
		if (!confirm(`Delete ${item.name}?`)) return
		await pb.collection("db_inventory").delete(item.id)
		await load()
	}

	async function setAlertStatus(alert: DBAlert, status: "acknowledged" | "resolved") {
		const payload: Record<string, unknown> = { status }
		payload[status === "acknowledged" ? "acknowledged" : "resolved"] = new Date().toISOString()
		await pb.collection("db_alerts").update(alert.id, payload)
		await load()
	}

	return (
		<div className="flex flex-col gap-4 pb-4">
			<div className="flex flex-wrap items-center justify-between gap-3">
				<div>
					<h1 className="text-xl font-semibold">Database Observability</h1>
					<p className="text-sm text-muted-foreground">
						Health, connections, locks, replication, and alerts collected every minute.
					</p>
				</div>
				<div className="flex gap-2">
					<Button variant="outline" size="sm" onClick={load}>
						<RefreshCwIcon className="size-4" />
						Refresh
					</Button>
					{admin && (
						<Dialog open={open} onOpenChange={setOpen}>
							<DialogTrigger asChild>
								<Button size="sm" onClick={startAdd}>
									<PlusIcon className="size-4" />
									Add database
								</Button>
							</DialogTrigger>
							<DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
								<DialogHeader>
									<DialogTitle>{editing ? "Edit database" : "Add database"}</DialogTitle>
								</DialogHeader>
								<div className="grid gap-3 sm:grid-cols-2">
									<div className="grid gap-1.5">
										<Label>Name</Label>
										<Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
									</div>
									<div className="grid gap-1.5">
										<Label>Application</Label>
										<Input
											value={form.application}
											onChange={(e) => setForm({ ...form, application: e.target.value })}
										/>
									</div>
									<div className="grid gap-1.5">
										<Label>Environment</Label>
										<Select
											value={form.environment}
											onValueChange={(v) => setForm({ ...form, environment: v })}
										>
											<SelectTrigger>
												<SelectValue />
											</SelectTrigger>
											<SelectContent>
												{["PROD", "STAGING", "UAT", "DEV", "DR", "OTHER"].map((env) => (
													<SelectItem key={env} value={env}>
														{env}
													</SelectItem>
												))}
											</SelectContent>
										</Select>
									</div>
									<div className="grid gap-1.5">
										<Label>Engine</Label>
										<Select value={form.engine} onValueChange={(v) => setForm({ ...form, engine: v })}>
											<SelectTrigger>
												<SelectValue />
											</SelectTrigger>
											<SelectContent>
												{["postgres", "mysql", "sqlserver", "oracle"].map((engine) => (
													<SelectItem key={engine} value={engine}>
														{engine}
													</SelectItem>
												))}
											</SelectContent>
										</Select>
									</div>
									<div className="grid gap-1.5">
										<Label>Host</Label>
										<Input value={form.host} onChange={(e) => setForm({ ...form, host: e.target.value })} />
									</div>
									<div className="grid gap-1.5">
										<Label>Port</Label>
										<Input
											type="number"
											value={form.port}
											onChange={(e) => setForm({ ...form, port: e.target.value })}
										/>
									</div>
									<div className="grid gap-1.5">
										<Label>Database</Label>
										<Input
											value={form.dbname}
											onChange={(e) => setForm({ ...form, dbname: e.target.value })}
										/>
									</div>
									<div className="grid gap-1.5">
										<Label>Username</Label>
										<Input
											value={form.username}
											onChange={(e) => setForm({ ...form, username: e.target.value })}
										/>
									</div>
									<div className="grid gap-1.5">
										<Label>Password {editing && "(leave blank to keep)"}</Label>
										<Input
											type="password"
											value={form.password}
											onChange={(e) => setForm({ ...form, password: e.target.value })}
										/>
									</div>
									<div className="grid gap-1.5">
										<Label>SSL mode</Label>
										<Select value={form.sslmode} onValueChange={(v) => setForm({ ...form, sslmode: v })}>
											<SelectTrigger>
												<SelectValue />
											</SelectTrigger>
											<SelectContent>
												{["disable", "require", "verify-full"].map((mode) => (
													<SelectItem key={mode} value={mode}>
														{mode}
													</SelectItem>
												))}
											</SelectContent>
										</Select>
									</div>
									<div className="grid gap-1.5">
										<Label>Owner</Label>
										<Input value={form.owner} onChange={(e) => setForm({ ...form, owner: e.target.value })} />
									</div>
									<div className="grid gap-1.5">
										<Label>HA mode</Label>
										<Input
											value={form.ha_mode}
											onChange={(e) => setForm({ ...form, ha_mode: e.target.value })}
										/>
									</div>
									<div className="grid gap-1.5">
										<Label>Cloud provider</Label>
										<Input
											value={form.cloud_provider}
											onChange={(e) => setForm({ ...form, cloud_provider: e.target.value })}
										/>
									</div>
									<div className="grid gap-1.5">
										<Label>Group</Label>
										<Input
											value={form.group_name}
											onChange={(e) => setForm({ ...form, group_name: e.target.value })}
										/>
									</div>
									<div className="flex items-center gap-2 pt-5">
										<Switch
											checked={form.enabled}
											onCheckedChange={(checked) => setForm({ ...form, enabled: checked })}
										/>
										<Label>Collection enabled</Label>
									</div>
								</div>
								<div className="flex justify-end gap-2">
									<Button variant="ghost" onClick={() => setOpen(false)}>
										Cancel
									</Button>
									<Button onClick={save} disabled={!form.name.trim() || !form.host.trim()}>
										Save
									</Button>
								</div>
							</DialogContent>
						</Dialog>
					)}
				</div>
			</div>

			<div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
				<SummaryCard title="Databases" value={String(inventory.length)} detail={`${upCount} reachable`} />
				<SummaryCard
					title="Availability"
					value={inventory.length ? `${Math.round((upCount / inventory.length) * 100)}%` : "-"}
					detail="Latest collection cycle"
				/>
				<SummaryCard
					title="Open alerts"
					value={String(alerts.length)}
					detail={`${criticalAlerts.length} critical`}
					alert={criticalAlerts.length > 0}
				/>
				<SummaryCard
					title="Replication lag max"
					value={`${Math.max(0, ...metrics.slice(0, 50).map((m) => m.replication_lag_sec || 0)).toFixed(0)}s`}
					detail="Most recent samples"
				/>
			</div>

			<Card>
				<CardHeader className="flex-row items-center justify-between gap-2">
					<CardTitle className="flex items-center gap-2 text-base">
						<DatabaseIcon className="size-4" /> Inventory &amp; health
					</CardTitle>
				</CardHeader>
				<CardContent className="overflow-x-auto">
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>Database</TableHead>
								<TableHead>Env</TableHead>
								<TableHead>Engine</TableHead>
								<TableHead>Target</TableHead>
								<TableHead>Status</TableHead>
								<TableHead>Connections</TableHead>
								<TableHead>Long q</TableHead>
								<TableHead>Blocked</TableHead>
								<TableHead>Lag</TableHead>
								<TableHead>Size</TableHead>
								<TableHead>Collected</TableHead>
								{admin && <TableHead />}
							</TableRow>
						</TableHeader>
						<TableBody>
							{inventory.length === 0 && (
								<TableRow>
									<TableCell colSpan={12} className="text-center text-sm text-muted-foreground">
										{loading ? "Loading..." : "No databases registered yet."}
									</TableCell>
								</TableRow>
							)}
							{inventory.map((item) => {
								const metric = latest.get(item.id)
								return (
									<TableRow key={item.id}>
										<TableCell className="font-medium">
											{item.name}
											<div className="text-xs text-muted-foreground">{item.application || "-"}</div>
										</TableCell>
										<TableCell className="text-xs">{item.environment || "-"}</TableCell>
										<TableCell className="text-xs">{item.engine || "-"}</TableCell>
										<TableCell className="text-xs">
											{item.host}:{item.port}
											<div className="text-muted-foreground">{item.dbname}</div>
										</TableCell>
										<TableCell>
											<Badge
												variant="secondary"
												className={metric?.up ? "bg-green-500/15 text-green-700 dark:text-green-400" : "bg-red-500/15 text-red-600 dark:text-red-400"}
											>
												{metric ? (metric.up ? "up" : "down") : "pending"}
											</Badge>
										</TableCell>
										<TableCell className="text-xs">
											{metric ? `${metric.connections_total}/${metric.connections_max} (${metric.connection_usage.toFixed(0)}%)` : "-"}
										</TableCell>
										<TableCell className="text-xs">{metric?.long_queries ?? "-"}</TableCell>
										<TableCell className="text-xs">{metric?.blocked_sessions ?? "-"}</TableCell>
										<TableCell className="text-xs">
											{metric ? `${metric.replication_lag_sec.toFixed(0)}s` : "-"}
										</TableCell>
										<TableCell className="text-xs">{metric ? formatBytes(metric.db_size_bytes, false) : "-"}</TableCell>
										<TableCell className="text-xs text-muted-foreground">
											{metric ? new Date(metric.collected).toLocaleTimeString(undefined, { timeZone: "Asia/Jakarta" }) : "-"}
										</TableCell>
										{admin && (
											<TableCell>
												<div className="flex gap-1">
													<Button variant="ghost" size="sm" onClick={() => startEdit(item)}>
														Edit
													</Button>
													<Button variant="ghost" size="sm" onClick={() => remove(item)}>
														Delete
													</Button>
												</div>
											</TableCell>
										)}
									</TableRow>
								)
							})}
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
								<TableHead />
							</TableRow>
						</TableHeader>
						<TableBody>
							{alerts.length === 0 && (
								<TableRow>
									<TableCell colSpan={7} className="text-center text-sm text-muted-foreground">
										No open database alerts.
									</TableCell>
								</TableRow>
							)}
							{alerts.map((alert) => {
								const item = inventory.find((i) => i.id === alert.inventory)
								return (
									<TableRow key={alert.id}>
										<TableCell className="text-sm">{item?.name ?? alert.inventory}</TableCell>
										<TableCell className="text-xs">{alert.metric}</TableCell>
										<TableCell>
											<Badge variant="secondary" className={statusColor[alert.severity] ?? ""}>
												{alert.severity}
											</Badge>
										</TableCell>
										<TableCell className="text-xs">{alert.value?.toFixed(2)}</TableCell>
										<TableCell>
											<Badge variant="secondary" className={statusColor[alert.status] ?? ""}>
												{alert.status}
											</Badge>
										</TableCell>
										<TableCell className="text-xs text-muted-foreground">
											{new Date(alert.triggered).toLocaleString(undefined, { timeZone: "Asia/Jakarta" })}
										</TableCell>
										<TableCell>
											{admin && (
												<div className="flex gap-1">
													<Button variant="ghost" size="sm" onClick={() => setAlertStatus(alert, "acknowledged")}>
														Ack
													</Button>
													<Button variant="ghost" size="sm" onClick={() => setAlertStatus(alert, "resolved")}>
														Resolve
													</Button>
												</div>
											)}
										</TableCell>
									</TableRow>
								)
							})}
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
