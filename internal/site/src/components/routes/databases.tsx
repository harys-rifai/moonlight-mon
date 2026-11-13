import { useLingui } from "@lingui/react/macro"
import { memo, useEffect, useState } from "react"
import NetworkMonitorsTableNew from "@/components/network-monitors-table/network-monitors-table"
import { FooterRepoLink } from "@/components/footer-repo-link"
import { useNetworkMonitors } from "@/lib/use-network-monitors"
import { pb } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { $router, navigate } from "@/components/router"
import { getPagePath } from "@nanostores/router"

// Common database ports recognized by the Databases page
const DATABASE_PORTS = new Set([5432, 3306, 6379, 1433, 1521, 50000, 3000, 5006, 5008])

export default memo(() => {
	const { t } = useLingui()
	const [isAuth, setIsAuth] = useState<boolean | null>(null)
	const { monitors, isLoading } = useNetworkMonitors({})

	// Check if user is authenticated
	useEffect(() => {
		setIsAuth(!!pb.authStore.isValid)
	}, [])

	if (isAuth === null) {
		return (
			<div className="flex h-full items-center justify-center">
				<Trans>Loading...</Trans>
			</div>
		)
	}

	if (!isAuth) {
		return (
			<div className="flex h-full flex-col items-center justify-center gap-4">
				<Trans>Please log in to view databases</Trans>
				<Button onClick={() => navigate(getPagePath($router, "settings", { name: "general" }))}>
					<Trans>Log In</Trans>
				</Button>
			</div>
		)
	}

	// Filter to database protocol monitors only (by port, since protocol is "tcp")
	// Database monitors are TCP checks and don't require agent version support.
	const visibleMonitors = monitors.filter((monitor) => DATABASE_PORTS.has(monitor.port))

	useEffect(() => {
		document.title = `${t`Databases`} / Moonlight apps`
	}, [t])

	return (
		<>
			<NetworkMonitorsTableNew
				monitors={visibleMonitors}
				isLoading={isLoading}
				title={t`Databases`}
				emptyMessage={t`No databases found.`}
			/>
			<FooterRepoLink />
		</>
	)
})
