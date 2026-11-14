import { PlusIcon } from "lucide-react"
import { useEffect, useState } from "react"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { pb } from "@/lib/api"

const ENGINES = [
	{ value: "postgres", label: "PostgreSQL", port: "5432" },
	{ value: "mysql", label: "MySQL", port: "3306" },
	{ value: "mariadb", label: "MariaDB", port: "3306" },
	{ value: "redis", label: "Redis", port: "6379" },
	{ value: "sqlserver", label: "SQL Server", port: "1433" },
	{ value: "oracle", label: "Oracle", port: "1521" },
	{ value: "db2", label: "DB2 (TCP monitor only)", port: "50000" },
]

const emptyForm = {
	name: "",
	engine: "postgres",
	environment: "DEV",
	host: "localhost",
	port: "5432",
	dbname: "postgres",
	username: "postgres",
	password: "",
	sslmode: "disable",
	owner: "",
	enabled: true,
}

/** Admin dialog to register a database collector target used by monitor detail. */
export default function AddDatabaseTarget({
	onSaved,
	preset,
}: {
	onSaved?: () => void
	preset?: { name?: string; host?: string; port?: number; engine?: string }
}) {
	const [open, setOpen] = useState(false)
	const [form, setForm] = useState({ ...emptyForm })
	const [saving, setSaving] = useState(false)
	const [error, setError] = useState("")

	useEffect(() => {
		if (!open || !preset) return
		const engine = preset.engine && ENGINES.some((item) => item.value === preset.engine) ? preset.engine : "postgres"
		const presetPort = ENGINES.find((item) => item.value === engine)?.port
		setForm({
			...emptyForm,
			name: preset.name ?? "",
			engine,
			host: preset.host ?? emptyForm.host,
			port: preset.port ? String(preset.port) : (presetPort ?? emptyForm.port),
		})
	}, [open, preset])

	async function save() {
		setSaving(true)
		try {
			await pb.collection("db_inventory").create({
				name: form.name.trim(),
				engine: form.engine,
				environment: form.environment,
				host: form.host.trim(),
				port: Number(form.port) || 0,
				dbname: form.dbname.trim(),
				username: form.username.trim(),
				password: form.password,
				sslmode: form.sslmode,
				owner: form.owner.trim(),
				enabled: form.enabled,
			})
			setOpen(false)
			setForm({ ...emptyForm })
			setError("")
			onSaved?.()
		} catch (err) {
			setError((err as Error)?.message ?? "Failed to save target")
		} finally {
			setSaving(false)
		}
	}

	return (
		<Dialog open={open} onOpenChange={setOpen}>
			<DialogTrigger asChild>
				<Button variant="outline" size="sm">
					<PlusIcon className="size-4" />
					Add DB target
				</Button>
			</DialogTrigger>
			<DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
				<DialogHeader>
					<DialogTitle>Add database target</DialogTitle>
				</DialogHeader>
				<div className="grid gap-3 sm:grid-cols-2">
					<div className="grid gap-1.5">
						<Label>Name</Label>
						<Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
					</div>
					<div className="grid gap-1.5">
						<Label>Engine</Label>
						<Select
							value={form.engine}
							onValueChange={(value) => {
								const preset = ENGINES.find((item) => item.value === value)
								setForm({
									...form,
									engine: value,
									port: preset?.port ?? form.port,
									dbname: value === "redis" ? "" : form.dbname,
								})
							}}
						>
							<SelectTrigger>
								<SelectValue />
							</SelectTrigger>
							<SelectContent>
								{ENGINES.map((item) => (
									<SelectItem key={item.value} value={item.value}>
										{item.label}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
					</div>
					<div className="grid gap-1.5">
						<Label>Environment</Label>
						<Select value={form.environment} onValueChange={(value) => setForm({ ...form, environment: value })}>
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
						<Label>Host</Label>
						<Input value={form.host} onChange={(e) => setForm({ ...form, host: e.target.value })} />
					</div>
					<div className="grid gap-1.5">
						<Label>Port</Label>
						<Input type="number" value={form.port} onChange={(e) => setForm({ ...form, port: e.target.value })} />
					</div>
					<div className="grid gap-1.5">
						<Label>{form.engine === "oracle" ? "Service name" : "Database"}</Label>
						<Input value={form.dbname} onChange={(e) => setForm({ ...form, dbname: e.target.value })} />
					</div>
					<div className="grid gap-1.5">
						<Label>Username</Label>
						<Input value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} />
					</div>
					<div className="grid gap-1.5">
						<Label>Password</Label>
						<Input
							type="password"
							value={form.password}
							onChange={(e) => setForm({ ...form, password: e.target.value })}
						/>
					</div>
					<div className="grid gap-1.5">
						<Label>SSL mode (PostgreSQL)</Label>
						<Select value={form.sslmode} onValueChange={(value) => setForm({ ...form, sslmode: value })}>
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
					<div className="flex items-center gap-2 pt-5">
						<Switch checked={form.enabled} onCheckedChange={(checked) => setForm({ ...form, enabled: checked })} />
						<Label>Collection enabled</Label>
					</div>
				</div>
				<div className="flex items-center justify-between gap-2">
					<p className="text-xs text-muted-foreground">{error}</p>
					<div className="flex gap-2">
						<Button variant="ghost" onClick={() => setOpen(false)}>
							Cancel
						</Button>
						<Button onClick={save} disabled={saving || !form.name.trim() || !form.host.trim()}>
							Save
						</Button>
					</div>
				</div>
			</DialogContent>
		</Dialog>
	)
}
