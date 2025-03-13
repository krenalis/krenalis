const esbuild = require('esbuild');
const fs = require('fs');

esbuild
	.build({
		entryPoints: ['javascript-snippet.js'],
		minify: true,
		format: 'iife',
		target: 'es5',
		write: false,
	})
	.then((result) => {
		const text = result.outputFiles[0].text;
		let transformed = text.replace(/meergo\.load\([^)]*\)/, '\n\t$&;\n\t');
		const snippet = `export const SNIPPET = \`<script>\n\t${transformed}</script>\`;\n`;
		fs.writeFileSync('src/constants/javascriptSnippet.ts', snippet);
	})
	.catch((err) => {
		console.error(err);
	});
