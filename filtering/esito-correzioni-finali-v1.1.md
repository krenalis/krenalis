# Krenalis — Correzioni finali delle proposte riconciliate 1.1

**Data:** 27 ottobre 2026. **Stato:** proposte finali da approvare, non ancora approvate né congelate. **Baseline:** `typed-filter-values@d69e61681b2da531bbd9fdd067c825b589338f97`.

## File interessati e correzioni

Sono aggiornate soltanto le proposte revisionate [06](06-specifica-semantica-riconciliata-v1.1.md), [07](07-ast-canonico-riconciliato-v1.1.md) e [08](08-ast-canonico-estratto-v1.1.md), oltre a questo nuovo rapporto. Le versioni storiche, le prime proposte 1.1 e i rapporti precedenti restano invariati.

| Correzione | Sezioni interessate | Risultato |
|---|---|---|
| N1 — Limiti AND/OR | 06 §§7.5, 28.19, 30 e changelog; 07 §§1.1, 6.4, 14.2, 15, 15.1, 18, 19, F1 e F4 | Massimo quattro **livelli di gruppi**, radice inclusa, e 100 nodi discendenti, radice esclusa. Grammatica e budget dei corpi invariati. |
| N2 — I096 | 06 §17.3 | Esplicitata la precedenza affidabile anche a parità di timestamp nei quattordici giorni precedenti l’upgrade. Tabella e convenzioni generali degli intervalli invariate. |
| N3 — JSON raw e valuta business | Nuovo 06 §28.20; rinvii in 07 §§4.1.2, 21.4 e F2 | Su JSON Null presente, il confronto raw con String "EUR" è FALSE e l’assenza qualificante è MATCH; il confronto governato NativeCurrency è UNKNOWN e l’assenza è UNKNOWN. Fonte e contratto disponibili, osservazione completa; costruzioni non intercambiabili. |
| N4 — Fonti ed executor | 06 §8.2; 07 §§14.4, 18 e F6 | Significato e requisiti informativi definiti per ogni combinazione ammessa in Unified, indipendenti dall’executor. Informazione insufficiente distinta da capacità strutturalmente indisponibile. Nessun nuovo registry Record o correzione retroattiva. |
| Estratto 08 | Intero documento, modifiche derivate in F1, F2, F4 e F6 | Rigenerato dalla sezione finale del 07, senza contenuto autonomo. |

## Controlli documentali

- **63 controlli statici superati:** modifiche limitate alle sezioni elencate, nuovi rinvii risolti e blocchi Markdown bilanciati.
- Tutti i **94 blocchi di grammatica ed esempi di codice del 07** sono identici alla proposta revisionata di partenza. Nessuna modifica all’architettura V/E/K/P/T.
- Catalogo chiuso dei **26 operatori**, relativa tabella normativa, payload e validazione invariati.
- Rimossi i riepiloghi ambigui che potevano far intendere quattro gruppi totali; budget principali e locali distinti.
- Verificati i prerequisiti e gli esiti della fixture raw/governata, i rinvii a 06 §28.20 e la distinzione fra UNKNOWN e capability indisponibile. Nessuna nuova contraddizione rilevata nei punti modificati.
- 08 è identico **byte per byte** alla sezione `# Canonical AST — Phase 4A Final Model` del 07. SHA-256 dell’estratto: `7e241deb2bc4e27eb8e200d24f3e1f89545965707e49233f693cd8cdeb18cb8a`.
- Integrità dei file preesistenti non interessati controllata mediante SHA-256, inclusi benchmark 03, audit, versioni storiche 06/07/08 e prime proposte 1.1.

Questa è una verifica documentale: non sono stati eseguiti test applicativi o sui warehouse, né modificati codice applicativo o executor. La verifica su Snowflake reale resta separata. Nessuna attività della Fase 4B e nessun commit; i documenti restano proposte in attesa di approvazione esplicita.
