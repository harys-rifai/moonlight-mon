import { useLingui } from "@lingui/react/macro"
import { memo, useEffect } from "react"
import NetworkMonitorsTableNew from "@/components/network-monitors-table/network-monitors-table"
import { FooterRepoLink } from "@/components/footer-repo-link"
import { useNetworkMonitors } from "@/lib/use-network-monitors"
import { $allSystemsById } from "@/lib/stores"
import { supportsNetworkMonitors } from "@/lib/utils"
import { useStore } from "@nanostores/react"

// Common database ports recognized by the Databases page
const DATABASE_PORTS = new Set([5432, 3306, 6379, 1433, 1521, 50000, 3000, 5006, 5008])

export default memo(() => {
	const { t } = useLingui()
	const { monitors, isLoading } = useNetworkMonitors({})
	const systems = useStore($allSystemsById)

	// Filter to database protocol monitors only (by port, since protocol is "tcp")
	// Database monitors are TCP checks and don't require agent version support.
	const visibleMonitors = monitors.filter((monitor) => {
		return DATABASE_PORTS.has(monitor.port)
	})

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
