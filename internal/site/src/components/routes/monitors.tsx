import { useLingui } from "@lingui/react/macro"
import { memo, useEffect, useState } from "react"
import NetworkMonitorsTableNew from "@/components/network-monitors-table/network-monitors-table"
import { FooterRepoLink } from "@/components/footer-repo-link"
import { useNetworkMonitors } from "@/lib/use-network-monitors"
import { pb } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { $router, navigate } from "@/components/router"
import { getPagePath } from "@nanostores/router"

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
				<Trans>Please log in to view monitors</Trans>
				<Button onClick={() => navigate(getPagePath($router, "settings", { name: "general" }))}>
					<Trans>Log In</Trans>
				</Button>
			</div>
		)
	}

	useEffect(() => {
		document.title = `${t`Network Monitors`} / Moonlight apps`
	}, [t])

	return (
		<>
			<NetworkMonitorsTableNew monitors={monitors} isLoading={isLoading} />
			<FooterRepoLink />
		</>
	)
})
