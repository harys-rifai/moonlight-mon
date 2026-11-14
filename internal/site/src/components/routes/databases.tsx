import { useLingui, Trans } from "@lingui/react/macro"
import { memo, useEffect, useState } from "react"
import NetworkMonitorsTableNew from "@/components/network-monitors-table/network-monitors-table"
import { FooterRepoLink } from "@/components/footer-repo-link"
import { useNetworkMonitors } from "@/lib/use-network-monitors"
import { pb, isAdmin } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { $router, navigate } from "@/components/router"
import { getPagePath } from "@nanostores/router"
import { isDatabaseMonitor } from "@/lib/database-monitors"
import { useDbCollectorStats } from "@/lib/db-collector-stats"
import AddDatabaseTarget from "@/components/add-database-target"

export default memo(() => {
	const { t } = useLingui()
	const [isAuth, setIsAuth] = useState<boolean | null>(null)
	const { monitors, isLoading } = useNetworkMonitors({})
	useDbCollectorStats()

	useEffect(() => {
		setIsAuth(!!pb.authStore.isValid)
	}, [])

	useEffect(() => {
		document.title = `${t`Databases`} / Moonlight apps`
	}, [t])

	if (isAuth === null) {
		return (
			<div className="flex min-h-[50vh] items-center justify-center p-6 text-sm text-muted-foreground">
				<Trans>Loading...</Trans>
			</div>
		)
	}

	if (!isAuth) {
		return (
			<div className="flex min-h-[50vh] flex-col items-center justify-center gap-4 p-6 text-sm text-muted-foreground">
				<Trans>Please log in to view databases</Trans>
				<Button onClick={() => navigate(getPagePath($router, "settings", { name: "general" }))}>
					<Trans>Log In</Trans>
				</Button>
			</div>
		)
	}

	const visibleMonitors = monitors.filter(isDatabaseMonitor)

	return (
		<div className="flex flex-col gap-4 pb-4">
			{isAdmin() && (
				<div className="flex justify-end">
					<AddDatabaseTarget />
				</div>
			)}
			<NetworkMonitorsTableNew
				monitors={visibleMonitors}
				isLoading={isLoading}
				title={t`Databases`}
				emptyMessage={t`No databases found.`}
				showDatabaseType
			/>
			<FooterRepoLink />
		</div>
	)
})
