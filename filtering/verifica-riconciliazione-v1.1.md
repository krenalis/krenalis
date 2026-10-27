# Krenalis — Verifica della proposta di riconciliazione v1.1

**Data:** 26 ottobre 2026. **Stato:** verifica documentale e strutturale; proposta da revisionare e approvare. **Baseline riportata da Codex:** `typed-filter-values@d69e61681b2da531bbd9fdd067c825b589338f97`.

## Materiale prodotto

- **06 — Specifica semantica riconciliata v1.1 (proposta):** integra il contratto pubblico comune, la matrice dei 26 operatori esistenti, assenza e JSON null, gli operatori di stato, il rapporto Go/SQL, quattro livelli/100 nodi, i contesti autorizzati e nuovi esempi di regressione.
- **07 — Canonical AST riconciliato v1.1 (proposta):** rappresenta i 26 operatori in `RecordCondition` all'interno di V e delle foglie autorizzate, mantiene le altre quattro primitive, adotta `CanonicalBool<4>` alla radice, limita i corpi separatamente, preserva path raw e documento pubblico, aggiorna il raccordo di §18.
- **08 — Estratto v1.1 (proposta):** estrazione letterale del modello finale dal nuovo 07, non una specifica autonoma.

## Controlli eseguiti qui

| Controllo | Esito |
|---|---|
| File prodotti in UTF-8, senza trailer binari degli allegati originali | Passato |
| Catalogo interno `ExistingRecordOperator26`: 26 nomi tecnici distinti | Passato |
| Radice del 07: `CanonicalBool<4>` nel testo principale e nell'estratto | Passato |
| 06: quattro gruppi principali, massimo 100 discendenti, budget dei body separati | Passato |
| Nessuna precedente istruzione radice `CanonicalBool<3, ScopedRule>` rimasta | Passato |
| Estratto 08 identico alla sezione finale del 07 | Passato |
| Bilanciamento dei blocchi di codice Markdown | Passato |
| Benchmark 03, audit 10 e specifiche v1.0 originali modificati | No |
| Test applicativi o accesso al repository in questa sessione | **Non eseguiti** |

## Punti da rivedere prima dell'approvazione

1. Verificare con il codice `d69e6168` che le tabelle di operatori/operandi e le fixture proposte rispecchino ogni combinazione ammessa, compresi JSON dinamico, unari e contenitori. Una revisione documentale non equivale a test di runtime.
2. Verificare che il nuovo nodo interno `RecordCondition` sia una rappresentazione fedele di tutti i `FilterCondition` esistenti, senza richiedere metadata governati non disponibili ai filtri salvati o cambiare il risultato sui valori mancanti.
3. Precisare in Fase 4B wire format, persistenza, migrazioni e caricamento delle nuove `FilterRule`, nonché la distinzione fra canonical equality interna e `Where.Equal`.
4. Definire la coerenza fra query/list/count/preview/export Unified all'istante T, evitando che un `WHERE` elimini evidenze incerte necessarie alle quantificazioni comportamentali.
5. Trattare separatamente le divergenze residue ordinarie (presenza/null, Object appiattiti, collazioni, Float/NaN, precisione temporale) e la verifica remota Snowflake.

**Conclusione:** la proposta è pronta per una review tecnica e semantica indipendente. Non è una specifica normativa già approvata né una certificazione dell'implementazione Unified. La Fase 4B resta sospesa fino alla review e alla formalizzazione delle decisioni rimanenti.
