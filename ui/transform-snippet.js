import { readAll } from 'std/io/read_all.ts';
import { writeAll } from 'std/io/write_all.ts';

const source = await readAll(Deno.stdin);
let s = new TextDecoder().decode(source);
s = s.replace(/chichiAnalytics\.load\([^)]*\)/, '\n\t$&;\n\t');
const snippet = `export const SNIPPET = \`<script>\n\t${s}</script>\`;`;
const enc = new TextEncoder();
writeAll(Deno.stdout, enc.encode(snippet));
