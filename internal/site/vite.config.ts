import { defineConfig } from "vite"
import path from "path"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { lingui } from "@lingui/vite-plugin"

export default defineConfig({
	base: "./",
	define: {
		// Display timezone for the dashboard. Set APP_TIMEZONE in the repo .env to override.
		__APP_TIMEZONE__: JSON.stringify(process.env.APP_TIMEZONE || "Asia/Jakarta"),
	},
	plugins: [
		react({
			babel: {
				plugins: ["macros"],
			},
		}),
		lingui(),
		tailwindcss(),
	],
	esbuild: {
		legalComments: "external",
	},
	resolve: {
		alias: {
			"@": path.resolve(__dirname, "./src"),
		},
	},
})
