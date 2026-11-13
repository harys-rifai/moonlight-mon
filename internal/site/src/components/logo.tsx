import { useId } from "react"

export function Logo({ className }: { className?: string }) {
	const id = useId()

	return (
		<svg role="img" aria-label="Moonlight" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 50" className={className}>
			<defs>
				<linearGradient id={id} x1="0%" y1="0%" x2="100%" y2="100%">
					<stop offset="0%" style={{ stopColor: "#747bff" }} />
					<stop offset="100%" style={{ stopColor: "#24eb5c" }} />
				</linearGradient>
			</defs>
			<text
				x="0"
				y="35"
				fontFamily="Inter, sans-serif"
				fontWeight="700"
				fontSize="36"
				fill={`url(#${id})`}
			>
				Moonlight
			</text>
		</svg>
	)
}
