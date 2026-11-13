import { useLingui, Trans } from "@lingui/react/macro"
import { memo, useEffect, useState } from "react"
import NetworkMonitorsTableNew from "@/components/network-monitors-table/network-monitors-table"
import { FooterRepoLink } from "@/components/footer-repo-link"
import { useNetworkMonitors } from "@/lib/use-network-monitors"
import { pb } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { $router, navigate } from "@/components/router"
import { getPagePath } from "@nanostores/router"

const DATABASE_PORTS = new Set([5432, 3306, 6379, 1433, 1521, 50000, 3000, 5006, 5008])

export default memo(() => {
	const { t } = useLingui()
	const [isAuth, setIsAuth] = useState<boolean | null>(null)
	const { monitors, isLoading } = useNetworkMonitors({})

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

	const visibleMonitors = monitors.filter((monitor) => DATABASE_PORTS.has(monitor.port))

	return (
		<div className="flex flex-col gap-4 pb-4">
			<NetworkMonitorsTableNew
				monitors={visibleMonitors}
				isLoading={isLoading}
				title={t`Databases`}
				emptyMessage={t`No databases found.`}
			/>
			<FooterRepoLink />
		</div>
	)
})
