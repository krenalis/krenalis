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
		let transformed = text.replace(/meergo\.load\([^)]*\)/, '\n  $&;\n  ');
		const snippet = `export const SNIPPET = \`<script>\n  ${transformed}</script>\`;\n\nexport const DOCUMENTATION_LINK = 'https://www.krenalis.com/docs/ref/admin/javascript-sdk';`;
		fs.writeFileSync('src/constants/snippets/javascript.ts', snippet);
	})
	.catch((err) => {
		console.error(err);
	});
