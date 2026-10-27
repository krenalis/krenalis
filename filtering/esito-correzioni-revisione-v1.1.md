# Krenalis — Esito delle correzioni alla proposta riconciliata v1.1

**Data:** 27 ottobre 2026. **Stato:** documenti revisionati, non ancora approvati. **Baseline di riferimento riportata:** `typed-filter-values@d69e61681b2da531bbd9fdd067c825b589338f97`. Non è stato consultato o modificato il repository durante questa revisione.

## Correzioni applicate

1. **07 §15 — Limiti della radice:** corretta la tabella da `Nesting root <=3` a `<=4`, coerentemente con `CanonicalBool<4, …>` e la rappresentazione pubblica; esplicitato il limite di 100 nodi discendenti, radice esclusa, da verificare **prima** della canonicalizzazione, senza confonderlo con i budget interni delle nuove primitive.
2. **07 §§2.1, 4.1.2, 14.5; F2/F6 — Validazione record:** introdotta l'attestazione tecnica `RecordAdmissibility`, distinta da `Proof` delle definizioni governate. La validazione di `RecordCondition` riutilizza schema, path, tipo, operatori e operandi del record filtering esistente; non richiede nuove revisioni semantiche, mapping o registry governato. Precisato anche nel modello finale estratto.
3. **07 — Riferimenti normativi:** resi espliciti tutti i rinvii alla **proposta 06 v1.1**. La versione 06 v1.0 è indicata come vigente fino all'approvazione della revisione 1.1. Sono state eliminate le menzioni non qualificate a “Frozen” e chiarito che il modello descritto è ancora una proposta.
4. **06 §29.4 e dichiarazione conclusiva:** le scelte su formato e persistenza si risolvono durante la progettazione della **Fase 4B**, prima di implementare le funzionalità che ne dipendono. Eliminato il riferimento finale improprio a un modello 1.1 già congelato.
5. **08 — Estratto:** rigenerato *testualmente* dalla sezione `# Canonical AST — Phase 4A Final Model` del 07 revisionato, senza contenuto indipendente.

## Verifiche documentali eseguite

- Presenza e unicità dei **26 operatori** nel catalogo chiuso `ExistingRecordOperator26`.
- Presenza di `CanonicalBool<4, ScopedRule>` e della regola di profondità root `<=4`; assenza del precedente `Nesting root <=3`.
- Presenza del limite pubblico dei 100 discendenti e sua applicazione prima delle riscritture canoniche.
- Distinzione testuale e tipologica fra `RecordAdmissibility` e `Proof`.
- Nessun rinvio `Frozen` non qualificato nel 07 revisionato.
- Corrispondenza **esatta** dell'intero 08 con la sezione finale del 07 revisionato.
- Controllo del bilanciamento dei blocchi Markdown con triple backtick.

I controlli sopra sono statici sul testo e **non equivalgono a una nuova verifica del codice o dei warehouse**.

## Questioni rimandate senza modifica

Payload e formato pubblico delle nuove `FilterRule`; versionamento e persistenza; compatibilità dei dati installati; implementazione concreta dei reference e delle prove di validazione; canonical equality, hash e detection delle modifiche; coerenza a T di listing, count, preview ed export; verifiche residue sugli executor, compreso Snowflake reale. Queste attività restano nel perimetro di progettazione della Fase 4B o di interventi successivi specifici, salvo ulteriore decisione normativa.

**Giudizio:** i rilievi puntuali della revisione indipendente sono stati recepiti nei documenti proposti. La loro approvazione normativa formale rimane distinta da questa verifica editoriale.
