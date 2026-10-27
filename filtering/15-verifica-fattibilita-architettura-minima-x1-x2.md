# Krenalis — Fase 4B: verifica di fattibilità dell’architettura minima, X1 e X2

**Stato:** rapporto sperimentale per revisione decisionale; nessuna approvazione dell’architettura esecutiva.

**Data:** 27 ottobre 2026.

**Baseline verificata:** `typed-filter-values@c23777219db49fbf6164b88b4e74ca55a5830f64`.

**Autorità semantica:** documenti 06/07/08 v1.2 approvati; decisioni documentate dal rapporto 13. Il rapporto 14 rimane in revisione; il rapporto 12 rimane una proposta non approvata.

## 1. Esito e confine della dimostrazione

**La B ridotta riesce a esprimere ed eseguire correttamente i sottoinsiemi X1/X2 provati, senza un piano relazionale autonomo. Non è ancora dimostrata sufficiente per tutte le combinazioni della semantica 1.2.** La generazione diretta non è l’ostacolo osservato: rimangono problemi negli algoritmi locali che determinano alternative compatibili, identity e quote, oltre alla loro scalabilità.

Sono stati realizzati due generatori SQL da modelli tipizzati sperimentali, un oracolo separato e un confronto A/B con **una sola implementazione**. PostgreSQL ha restituito l’esito atteso in **1.022/1.022 casi**; **1.021** sono confrontati con l’oracolo esaustivo del relativo dominio, per **3.891 storie/completamenti/basi visitati**. Il caso sul dominio Int64 è distinto: risultato verificato sul database e giustificazione analitica, senza enumerare 2⁶⁴ valori. Passano inoltre sette test unitari, otto rifiuti espliciti di input fuori dal sottoinsieme sperimentale e cinque prove di crescita. Queste categorie non vengono sommate per gonfiare il numero dei casi semantici.

Due risultati sono particolarmente rilevanti:

- **E:** il controesempio preferredCategory A/B del rapporto 14 è risolto conservando un unico fatto condiviso. Sono provati anche confronti ordinati su griglie numeriche, selezioni con parità, N=10 e un’undicesima occurrence. Il generatore non enumera tutti gli ordini o tutti i completamenti della storia.
- **K/share:** membership e body rimangono separati; N2 rimane composizionale. La share del 100% con una presenza certa e una opzionale, entrambe qualificanti, è TRUE. Per altre uguaglianze, fondere gli stati della base in un unico intervallo perde informazione: con basi che producono solo 1 e 1/2, share=3/4 è FALSE. Questo non autorizza a restringere l’inviluppo N2 dei body UNKNOWN.

La share ponderata ha un limite ulteriore, riprodotto: raggiungere una quota esatta con occurrence opzionali può richiedere raggiungibilità di somme di sottoinsiemi. Il prototipo la risolve per istanze piccole, ma il numero di stati può crescere esponenzialmente. Un IR relazionale non eliminerebbe quel problema. Non si propone UNKNOWN come ripiego, né un nuovo limite normativo per nasconderlo.

**Snowflake non è stato eseguito.** Questo rapporto non dimostra compatibilità reale PostgreSQL/Snowflake, un compilatore completo da tutto il Canonical AST, né prestazioni di query multiprofilo.

## 2. Baseline, fonti e isolamento

HEAD e branch corrispondono alla baseline richiesta. All’avvio non vi erano differenze tracked o staged; erano presenti file non tracciati preesistenti, fra cui il rapporto 14. Sono stati conservati. Prima della consegna è stata confrontata l’intera fotografia `git status --porcelain=v1 --untracked-files=all`; prima dell’aggiunta del solo rapporto 15 risultava identica a quella iniziale.

Sono stati verificati nuovamente i passaggi normativi necessari: 06 §§10.3–10.5, 11, 12.6, 19.3–19.4; 07 §§4.1–4.3, 7.3–7.5, 8.1–8.4, 9.1 e modello finale. Restano vincolanti selezione prima di verify, nessun fallback, appartenenza incerta su occurrence registrate, dipendenze interne e quantificazione N2. Il mondo chiuso non introduce occurrence ipotetiche. Il rapporto 14 §6.2 fornisce il controesempio principale; §§6.3–6.4 distinguono candidate base e share.

Riferimenti principali: [06 §11](/Users/marco/src/krenalis/filtering/06-specifica-semantica-riconciliata-v1.2.md:457), [06 §12.6](/Users/marco/src/krenalis/filtering/06-specifica-semantica-riconciliata-v1.2.md:577), [06 §19.4](/Users/marco/src/krenalis/filtering/06-specifica-semantica-riconciliata-v1.2.md:1049), [07 §7.3](/Users/marco/src/krenalis/filtering/07-ast-canonico-riconciliato-v1.2.md:777), [07 §8.1](/Users/marco/src/krenalis/filtering/07-ast-canonico-riconciliato-v1.2.md:869), [07 §9.1](/Users/marco/src/krenalis/filtering/07-ast-canonico-riconciliato-v1.2.md:965).

Sono stati riletti anche `warehouses.RowQuery` e `Expr` e gli ingressi `Query` dei due driver: [warehouses.go:386](/Users/marco/src/krenalis/warehouses/warehouses.go:386), [warehouses.go:796](/Users/marco/src/krenalis/warehouses/warehouses.go:796), [postgresql/query.go:17](/Users/marco/src/krenalis/warehouses/postgresql/query.go:17), [snowflake/query.go:17](/Users/marco/src/krenalis/warehouses/snowflake/query.go:17). `RowQuery` rappresenta tabella/join/where/ordinamento/paginazione; non contiene le relazioni di alternative qui necessarie. I prototipi non sono stati forzati in quei tipi. L’integrazione con un ingresso warehouse e il riuso dei renderer Record rimangono lavori successivi: usare `psql` non prova che `Warehouse.Query` o `QueryReadOnly` possano essere riutilizzati senza adattamenti.

Tutto il codice sperimentale e gli output si trovano fuori dal repository:

`/private/tmp/krenalis-x1-x2-15-93okkmyz/`

Ambiente effettivo: macOS 27.0.1 arm64, Python 3.13.2, Docker 29.9.2; immagine locale `postgres:18.6-alpine3.24`, PostgreSQL 18.6 aarch64. Image ID: `sha256:d7a8005067f55a2f8ea369e767545fd467b1943a4f0f1a48b7dc44f06c366109`.

Ogni esecuzione crea un container nuovo, senza rete (`--network none`), porte pubblicate o volumi, con 512 MiB e due CPU. Non vi sono dati o credenziali condivisi. Le query sono read-only e hanno `statement_timeout=30s`; i dati sintetici sono relazioni `VALUES`. Il runner attende il server finale, non soltanto il server temporaneo di inizializzazione, e rimuove il container al termine. La rimozione è verificata nell’esecuzione finale.

Si assume la vista coerente richiesta dal mandato. Nessun protocollo di generazioni, watermark, snapshot persistenti o materializzazione è introdotto. Identity Resolution/pubblicazione Profiles, writer/attribuzione KPID e destination matching rimangono lavori autonomi rinviati.

## 3. Codice eseguibile e indipendenza dell’oracolo

### 3.1 Artefatti

Il prototipo contiene sei file, 1.141 righe complessive, inclusi modelli, fixture, test, runner e oracolo. Il generatore è un solo file di 379 righe, con due ingressi `compile_e` e `compile_k`; non sono cinque motori o due compilatori A/B. Sono dataclass immutabili, enum e controlli runtime Python: **non costituiscono una prova di type safety del futuro codice Go**.

| File | Righe | SHA-256 dei sorgenti eseguiti |
|---|---:|---|
| [compiler.py](/private/tmp/krenalis-x1-x2-15-93okkmyz/prototype/compiler.py) | 379 | `56e053ccc70e8d74c427a86982165d61d278f92e0522238d579501e1b63692b6` |
| [fixtures.py](/private/tmp/krenalis-x1-x2-15-93okkmyz/prototype/fixtures.py) | 183 | `82450e501e404a27988d5a7dde7b0c0f612624cb1b6e3042838e449c2b28c729` |
| [model.py](/private/tmp/krenalis-x1-x2-15-93okkmyz/prototype/model.py) | 180 | `f629a21a162293cdd7f02c21da1fa7c0e186803b92458675b03fe47f6b4b603b` |
| [oracle.py](/private/tmp/krenalis-x1-x2-15-93okkmyz/prototype/oracle.py) | 186 | `2162034264a19fc1153650581503d6b19c625a0a55f5effa446deae09bf3251c` |
| [run.py](/private/tmp/krenalis-x1-x2-15-93okkmyz/prototype/run.py) | 160 | `39abbcc5ddc97bfdc4ad1c44b5f4570f90d3cf54a6586ab918ea5c904ab2948b` |
| [test_contracts.py](/private/tmp/krenalis-x1-x2-15-93okkmyz/prototype/test_contracts.py) | 53 | `a3894acee2af8df3a87e991558cb9744b4b80f0b1c8584e22251623f10a38ae7` |

`model.py` rappresenta il sottoinsieme necessario all’esperimento, non un nuovo formato pubblico o una sostituzione del Canonical AST. `E.selector` discrimina i selettori; i preset Last/Previous consumano il solo campo score e il proprio check, non un verify arbitrario. Per ridurre il solo harness, `K.weighted_occurrence_share` instrada anche le prove **V.WeightedShare**: in quel ramo le identity sono identity di occurrence e il contratto di grana è Occurrence. Non viene introdotta una share ponderata dei candidati K nella specifica.

`compiler.py` deriva le CTE dalla costruzione, dal catalogo dei fatti e dagli schemi sintetici. Cambiare selettore, N, predicato, dominio, quantificatore o dati attraversa gli stessi helper. Le CTE non sono state scritte caso per caso. La sola sorgente `VALUES` è generata dai dati del modello; soglie e valori non passano da float, gli identificatori SQL del micro-modello sono chiusi e le String sono quotate come literal.

Ingressi verificabili: [compile_e](/private/tmp/krenalis-x1-x2-15-93okkmyz/prototype/compiler.py:170), [compile_k](/private/tmp/krenalis-x1-x2-15-93okkmyz/prototype/compiler.py:321), [oracle_e](/private/tmp/krenalis-x1-x2-15-93okkmyz/prototype/oracle.py:83), [oracle_k](/private/tmp/krenalis-x1-x2-15-93okkmyz/prototype/oracle.py:157). Il compilatore sperimentale non include il validatore completo del Canonical AST; valida i vincoli del solo micro-modello ed esplicita i casi ancora fuori copertura. Non è una prova della futura pipeline di risoluzione e validazione pubblica.

Il SQL è però **PostgreSQL sperimentale**: include `numeric`, cast `::`, `LATERAL` e `json_build_object`. Le funzioni JSON finali sono soltanto il trasporto dei risultati del test. Non viene presentato come generatore comune a entrambi i dialetti già completato.

### 3.2 Oracolo

`oracle.py` importa soltanto il modello. Non importa il compilatore, il partizionamento in celle, le formule di rango o il calcolo SQL degli stati. Per E enumera i valori dei piccoli domini governati, i flag locali ignoti pertinenti alla qualificazione e gli ordini ammessi nelle parità; su ciascuna storia determina la base concreta, seleziona e poi verifica. Per K enumera le basi possibili, risolve/deduplica le identity finite e applica N2 ai singoli risultati ternari, senza completare globalmente le chiavi dei Refund per rafforzare il quantificatore.

Le **85 storie completamente determinate** di lunghezza 0–3 su categoria A/B × flag TRUE/FALSE, con timestamp strettamente ordinati e score noti, sono tutte enumerate e ciascuna è verificata sui cinque selettori: **425 casi EC**. I casi piccoli con incertezza aggiungono un confronto esaustivo dei completamenti del loro modello, non una previsione su dati casuali non specificati.

Per Latest10 con undici occurrence tutte a pari timestamp l’oracolo enumera gli undici sottoinsiemi selezionabili di dieci elementi. Le permutazioni interne al blocco non cambiano il medesimo predicato/quantificatore. Per Nth10 si visitano le undici identity che possono occupare quella posizione. È un quoziente esaustivo per quei test, non un’omissione di ordini pertinenti. I contatori “worlds” includono tali rappresentanti, non tutti gli 11! ordini equivalenti.

Un body esplicitamente UNKNOWN resta un esito ternario dove la semantica lo prescrive. In particolare, l’oracolo K **non** enumera completamenti comuni dei body per rimuovere l’incertezza vietata da N2. Questa distinzione impedisce di usare come oracolo una semantica più forte di quella approvata.

Sono conservati [modelli di ingresso](/private/tmp/krenalis-x1-x2-15-93okkmyz/release-evidence/models.json), [risultati per caso](/private/tmp/krenalis-x1-x2-15-93okkmyz/release-evidence/results.json), [rifiuti](/private/tmp/krenalis-x1-x2-15-93okkmyz/release-evidence/rejected.json), [riepilogo](/private/tmp/krenalis-x1-x2-15-93okkmyz/release-evidence/summary.json) e [SQL generato](/private/tmp/krenalis-x1-x2-15-93okkmyz/release-evidence/sql). L’appendice C incorpora l’archivio esatto dei sei sorgenti, per riprodurre gli esperimenti anche dopo la rimozione della directory temporanea. I transcript locali completi possono essere rigenerati; i risultati storici qui riportati sono quelli effettivamente ottenuti in questa esecuzione.

## 4. X1 — Derivazione delle selezioni compatibili E

### 4.1 Modello supportato e relazioni

L’esperimento opera su un profilo/owner alla volta. Ogni occurrence ha identity certa, timestamp noto, eventuale sequenza affidabile, categoria, campi numerici e un flag ternario. I timestamp sono tick ordinati sintetici, non una prova su fusi orari o calendario. A timestamp uguali, una sequenza nota definisce gruppi di ordine; senza sequenza nessun ID viene usato per sciogliere la parità.

Un fatto governato condiviso ha identity unica, dominio dichiarato e, se noto, valore. Lo stesso riferimento è mantenuto in tutte le occurrence. Sono supportati fino a due fatti indipendenti: enum String chiusi con uguaglianza/diversità oppure griglie numeriche esatte finite, con minimo, massimo e passo dichiarati. I confronti sono fra campo noto della occurrence e literal/fatto corrente condiviso. AND/OR compongono queste foglie nel sottoinsieme testato. Le stesse dipendenze possono comparire nel verify.

| Relazione | Grana / contenuto | Invariante |
|---|---|---|
| `occ` | Occurrence registrata | Identity unica; gruppo d’ordine debole `g` |
| `contexts` | Alternativa del medesimo insieme di fatti condivisi | Un valore/cella per fatto, conservato per l’intera selezione |
| `evaluated` | Contesto × occurrence × variante locale pertinente | Qualificazione e body valutati sulla stessa variante |
| `classified` | Contesto × occurrence | `must`, `can`, body condizionato alla presenza qualificante |
| Coppie / `bounds` | Selezione congiunta oppure stato di bordo del blocco | Nessun prodotto di posizioni indipendenti |
| `outcomes` | Esiti delle selezioni compatibili | Include FALSE se la posizione/blocco può mancare |

I flag locali sono completati solo quando incidono sulla qualificazione. Una stessa variante alimenta anche il body: non si separa “può qualificare” da “soddisfa verify quando qualifica”. Se una stessa variabile locale è dichiarata condivisa fra occurrence, il prototipo la rifiuta: va rappresentata e trattata come fatto condiviso, non come due flag indipendenti.

### 4.2 Oltre A/B: celle ordinate derivate nel SQL

Per un fatto numerico `v` su una griglia con passo `s`, il generatore raccoglie nel SQL tutti i comparandi noti usati dalle foglie riferite a quel fatto. Crea tagli interi con `floor(x/s)` e `ceil(x/s)`, aggiunge gli estremi del dominio, ordina i tagli e rappresenta anche ogni intervallo non vuoto fra tagli. Non vengono enumerate tutte le quantità possibili.

La giustificazione è circoscritta: fra due tagli nessun confronto ammesso campo-noto/fatto può cambiare verità; l’uguaglianza è distinta dalle due regioni ordinate. Le congiunzioni/disgiunzioni di tali foglie e il verify mantengono quindi un esito costante nella cella. Con due fatti dichiarati indipendenti si usa il prodotto delle rispettive celle. Non è una dimostrazione per vincoli arbitrari fra fatti, confronti fra quantità ignote di occurrence diverse, domini continui privi di un passo governato o altri predicati non partizionati.

`E19_int64_not_enumerated` usa l’intero dominio signed Int64: il database costruisce **11 contesti e 55 righe classificate**, non 2⁶⁴ valori. La storia contiene anche una occurrence con valore massimo del dominio, che garantisce una base non vuota per `x >= fatto`; il verify è sempre TRUE. Il PASS è analitico + esecuzione, non esaustivo. `E20_decimal_open_cells` usa passo 0,01 e comparandi −0,015 e 0,025: confronta nove completamenti dell’oracolo con otto celle SQL e restituisce TRUE. `E21_exact_decimal_pair` verifica una differenza esatta di 10⁻³⁷ su score maggiori di 2⁵³, senza conversione floating point.

### 4.3 Selettori e compatibilità

Per ogni contesto le occurrence qualificanti indipendenti hanno presenza necessaria `must` o possibile `can`. I gruppi `g` rispettano timestamp e ordine affidabile; una parità non viene resa totale artificialmente.

- **First/Last:** una occurrence possibile è selezionabile se non esistono qualificanti necessari strettamente prima/dopo. I pari possono essere disposti in modo compatibile.
- **Nth:** una occurrence possibile occupa N se il minimo numero necessario strettamente prima è minore di N e il numero possibile prima o nello stesso gruppo è almeno N. L’oracolo verifica l’esistenza degli ordini concreti.
- **Last/Previous:** entrambe le occurrence devono appartenere allo stesso contesto, essere distinte e selezionabili congiuntamente. Previous non può seguire Last; nessuna altra occurrence necessaria può trovarsi strettamente dopo Previous, salvo Last. Le altre eventuali occurrence pari possono essere poste prima di Previous. Questa condizione deriva la coppia; non si incrociano due elenchi di ranghi possibili.
- **Latest N:** si sceglie un possibile gruppo di bordo `g`, con `p<N` qualificanti successivi e `N-p` elementi del gruppo. I conteggi necessari/possibili per body TRUE/FALSE/NULL delimitano i successi del blocco, condizionati al medesimo bordo e allo stesso p. Non si enumerano tutte le permutazioni o tutti i sottoinsiemi di N eventi.

Per Latest N, siano `Rt/Rf` i TRUE/FALSE necessari successivi al bordo, `Pt/Pf/Pu` i possibili successivi, `Bt/Bf/Bu` i possibili nel gruppo di bordo e `k=N-p`. I bounds del numero di successi, nel modello di indipendenza locale dichiarato, sono:

```text
lo = max(Rt, p-Pf-Pu) + max(0, k-Bf-Bu)
hi = min(p-Rf, Pt+Pu) + min(k, Bt+Bu)
```

Sono ammessi solo p compresi fra numerosità necessaria e possibile della coda e bordi che forniscano k elementi. Le formule sono specifiche di questa selezione e del quantificatore del blocco. Non costituiscono un helper universale di bounds per K, E e share ponderate.

Se il numero di qualificanti necessari è inferiore alla posizione richiesta, si conserva anche l’alternativa di selezione assente, con esito FALSE. Se tutte le alternative esistenti hanno body FALSE, l’assenza possibile non causa da sola UNKNOWN: tutti gli esiti sono FALSE. La chiusura restituisce TRUE se tutti sono TRUE, FALSE se tutti sono FALSE, NULL negli altri casi. Non usa aggregati booleani che scartano NULL.

Il verify viene consumato **dopo** la selezione. Un Last con score 50 resta quello selezionato anche se uno precedente aveva score 200. Il preset Last/Previous non ripiega su una coppia più antica favorevole. La guardia del denominatore positivo può confutare una ratio anche con numeratore ignoto.

### 4.4 Controesempio obbligatorio, eseguito

Il fatto governato `profile.preferredCategory@r1` ha esattamente un valore in `{A,B}`, sconosciuto ma condiviso. Se il dominio ammettesse anche altre categorie, non sarebbe lecito escludere una selezione vuota: la chiusura del dominio è parte della fixture, non una supposizione dell’algoritmo.

| Occurrence, in ordine | Categoria | Score |
|---|---|---:|
| e1 | A | 90 |
| e2 | B | 140 |
| e3 | A | 100 |
| e4 | B | 150 |

Base: categoria uguale al fatto corrente. Test: score Last − score Previous ≥10.

| Contesto SQL | Last / Previous effettivi | Differenza | Esito |
|---|---|---:|---|
| A | e3 / e1 | 10 | TRUE |
| B | e4 / e2 | 10 | TRUE |

Risultato effettivo `E01_report14_shared_pair`: `{"value":true,"outcomes":2,"contexts":2,"classified":8}`. Le coppie sono state interrogate e salvate separatamente, non soltanto dedotte dal valore finale. Il SQL completo prodotto è nell’appendice A.

La variante sperimentale che perde il vincolo `a.ctx=p.ctx` produce anche e3/e2 (−40) ed e4/e3 (50): quattro esiti, risultato **NULL**. È un controesempio eseguibile alla conservazione delle dipendenze per sola classificazione indipendente dei ranghi. La B controllata rifiuta quell’assemblaggio prima del rendering; non “corregge” NULL dopo l’esecuzione.

### 4.5 Copertura E

| Caso eseguito | Esito PostgreSQL | Oracolo / motivazione |
|---|---|---|
| `E02_first` | TRUE | First, base qualificata prima del verify |
| `E02_last` | TRUE | Last sullo stesso fatto A/B |
| `E02_nth` | TRUE | Secondo qualificante per ciascuna alternativa |
| `E02_latest` | TRUE | Due ultimi qualificanti; guardia N rispettata |
| `E03_shared_body_dependency` | TRUE | Verify dipende dallo stesso fatto governato |
| `E04_shared_body_false` | FALSE | Verify certamente falso nelle selezioni compatibili |
| `E05_shared_body_unknown` | NULL | Campo decisivo ignoto nel verify |
| `E06_empty_last` | FALSE | Base vuota: posizione assente, non UNKNOWN |
| `E07_insufficient_latest` | FALSE | Meno di N occurrence |
| `E08_tie_last` | NULL | Due pari con body diversi, nessun tie-break su ID |
| `E09_tie_all_true` | TRUE | Identità del vincitore incerta ma test sempre vero |
| `E10_reliable_sequence` | FALSE | Sequenza affidabile scioglie la parità |
| `E11_no_fallback` | FALSE | Ultimo assoluto falso; nessun recupero del precedente favorevole |
| `E12_possible_last_false` | NULL | Ultimo soltanto possibile con verify falso |
| `E13_possible_last_true` | TRUE | Entrambi i possibili Last hanno verify vero |
| `E14_conditional_body_same_flag` | TRUE | Qualificazione e verify condividono il flag locale |
| `E15_pair_tie_unknown` | NULL | Due ordini possibili della coppia cambiano il test |
| `E16_pair_tie_same` | TRUE | SameValue resta determinato nella parità |
| `E18_numeric_shared_verify` | TRUE | Confronto ordinato e verify sullo stesso valore numerico |
| `E22_two_shared_facts` | TRUE | Due fatti, di cui uno numerico; 14 completamenti dell’oracolo |
| `E23_nth10_with11` | TRUE | Posizione 10 con undici occurrence ordinate |
| `E24_latest10_excludes11th` | TRUE | Il blocco esclude l’undicesima rispetto alla coda |
| `E25_latest10_tie11` | NULL | Undici sottoinsiemi compatibili di dieci elementi |
| `E26_nth10_tie11` | TRUE | Undici possibili identity alla posizione 10 |
| `E30_pair_no_fallback` | FALSE | La coppia più recente fallisce, una più antica sarebbe favorevole |
| `E32_ratio` | TRUE | Rapporto del medesimo campo, denominatore positivo |
| `E33_ratio_guard_false_unknown_numerator` | FALSE | Guardia falsa prevale su numeratore ignoto |
| `E34_optional_tenth_guard` | NULL | L’esistenza della decima occurrence è incerta |
| `E35_cardinality_between` | TRUE | Cardinalità inclusiva sullo stesso Latest N |

I 54 esempi E mirati sono integrati da 180 casi pseudo-casuali piccoli a seed fisso `15012026`, 80 casi con due fatti condivisi e qualificatori composti e i 425 casi completamente determinati EC. Le storie casuali hanno fino a cinque occurrence; i casi con due fatti fino a quattro. Posizioni 1–3 sono variate nei casi piccoli; il limite 10 e l’undicesima occurrence hanno fixture dedicate. Non viene dichiarato un test esaustivo di tutti i dati e di ogni posizione 1–10.

## 5. X2 — Candidate base, N2 e share

### 5.1 Membership distinta dal body

La sorgente mantiene `namespace`, identity pertinente, `membership` TRUE/FALSE/NULL e `body` TRUE/FALSE/NULL. FALSE di membership esclude la presenza; NULL rende opzionale quella identity; NULL del body non la elimina dalla base quando presente. Si deduplica per contesto/namespace/identity **prima** della quantificazione. Una prova certa di appartenenza prevale su prove soltanto possibili della stessa identity.

Nel sottoinsieme `no_refund`, il body individuale “nessun rimborso di quest’ordine” è FALSE se esiste un Refund con chiave corrispondente certa; altrimenti NULL se esiste un Refund registrato con chiave pertinente ignota; altrimenti TRUE. Due righe duplicate di Refund non creano altri ordini.

Con due ordini certi O1/O2 e un unico Refund con `orderId` ignoto, entrambi i body sono NULL. Il compilatore conserva `S=0,F=0,U=2,D=2`: **Some è NULL**. Non completa la chiave del Refund con un valore comune per provare che almeno un ordine è certamente senza rimborso. L’inferenza più forte sarebbe possibile nel dominio, ma è esclusa dalla scelta approvata N2. Some, None, All, Cardinality e Share sono tutti verificati su questo esempio.

Il prototipo tratta anche identity incerte con uno o due fatti enum finiti: risolve ogni contesto compatibile, deduplica e solo dopo quantifica. Con una occurrence di O1 e una con identity in `{O1,O2}`, la base ha uno o due ordini: count≥2 è NULL, mentre share=1 con tutti i body TRUE è TRUE. Se due occurrence leggono il **medesimo** fatto identity ignoto, ricevono lo stesso valore nel contesto; non vengono inventate due scelte indipendenti.

Questa prova non risolve una identity ignota su un dominio aperto, alias non risolti, identity multivalore o qualunque correlazione fra membership di identity distinte. Il modello dichiara indipendenti le membership opzionali una volta risolto il contesto. Provenienze condivise non rappresentabili sono rifiutate, non degradate a NULL. Body o pesi incompatibili su righe potenzialmente fuse nella stessa identity sono anch’essi fuori dal sottoinsieme.

### 5.2 Stati della base e quantificazione composizionale

Per ciascun contesto si contano le identity necessarie nei tre stati di body e le identity opzionali negli stessi stati. Con `RS,RF,RU` necessari e `OT,OF,OU` opzionali, il SQL deriva:

```text
S = RS + t, 0 <= t <= OT
F = RF + f, 0 <= f <= OF
U = RU + u, 0 <= u <= OU
D = S + F + U
```

Ogni stato rappresenta una classe di basi possibili nel sottoinsieme dichiarato. Non si enumerano 2^n sottoinsiemi: gli stati sono al più `(OT+1)(OF+1)(OU+1)` per contesto. Questo resta un costo potenzialmente elevato, non una promessa di scalabilità generale.

**In ciascuno stato** si applicano le formule N2: count `[S,S+U]`, Some/None, All con minimo e F confutante, Cardinality AtLeast/AtMost/Exactly/BetweenInclusive, Share `[S/D,(S+U)/D]` con guardie. Poi si chiudono gli stati compatibili con consenso ternario. In questo modo non si confonde l’incertezza della base con la conservatività normativa dei body.

Per una base certamente vuota: Some FALSE, None TRUE, count=0, All e Share FALSE per guardia. Una base soltanto possibile non dà automaticamente NULL: per esempio un massimo di un candidato con minimo due fa fallire certamente la guardia. All con un body certamente FALSE rimane FALSE anche se altri candidati sono opzionali o ignoti.

Gli endpoint della share sono confrontati mediante prodotti incrociati di interi/Decimal esatti, non mediante divisione arrotondata. Numeratore e denominatore derivano dalle **stesse** identity e dallo **stesso** stato.

### 5.3 Quote della base e inviluppo N2

| Dati | Richiesta | Esito verificato | Motivo |
|---|---|---|---|
| O1 certo TRUE; O2 opzionale TRUE | Share=1, minimo 1 | TRUE (`K02`) | Quote 1/1 e 2/2, guardia già certa |
| O1 certo TRUE; O2 opzionale FALSE | Share=1 | NULL (`K03`) | Quote 1 e 1/2 |
| Stessa base incerta, body tutti determinati | Share=3/4 | FALSE (`K04`) | Né 1 né 1/2 soddisfano il test; non si fonde la base in `[1/2,1]` |
| Base certa: O1 TRUE e O2 UNKNOWN | Share=3/4 | NULL (`K18`) | N2 impone l’inviluppo composizionale `[1/2,1]`; non lo si restringe usando la discretezza dei conteggi |

L’ultima riga è intenzionale. Applicare alle incertezze dei body lo stesso algoritmo esatto delle basi cambierebbe la semantica N2. Viceversa, estendere automaticamente l’inviluppo N2 a tutte le basi compatibili indebolirebbe risultati determinabili come `K04`. Sono due livelli diversi, con la stessa base mantenuta per part/whole.

### 5.4 Casi K eseguiti

| Caso eseguito | Esito PostgreSQL | Oracolo / motivazione |
|---|---|---|
| `K01_N2_some` | NULL | Due ordini certi; un Refund con chiave ignota; S=0,F=0,U=2 |
| `K02_optional_all_qualifying` | TRUE | Guardia certa e quota 100% in entrambe le basi |
| `K03_optional_denominator` | NULL | Il candidato non qualificante cambia il denominatore |
| `K04_ratio_gap` | FALSE | 3/4 non raggiungibile dalle due basi con body determinati |
| `K05_refund_duplicates` | TRUE | Due candidate identity; Refund duplicati non moltiplicano gli ordini |
| `K06_empty_none` | TRUE | Base certamente vuota; None vero |
| `K06_empty_all` | FALSE | Base certamente vuota; All non vacuo falso |
| `K07_uncertain_base_some` | NULL | Un candidato opzionale qualificante |
| `K08_body_unknown_all` | NULL | Base determinata, body UNKNOWN |
| `K09_minimum_always_fails` | FALSE | Minimo due, al massimo un candidato |
| `K10_minimum_depends_base` | NULL | Minimo due dipende dall’opzionale |
| `K11_certain_false_all` | FALSE | Un body FALSE confuta All |
| `K12_duplicate_optional_identity` | FALSE | Non si contano due volte prove opzionali della medesima identity |
| `K13_required_dominates_optional_duplicate` | TRUE | Appartenenza certa prevale sulla stessa prova opzionale |
| `K15_identity_dedup_uncertain` | NULL | La seconda identity può coincidere con O1 oppure essere O2 |
| `K16_identity_dedup_determined_test` | TRUE | Pur variando D, tutti i candidati qualificano |
| `K17_shared_identity_fact` | NULL | Due righe leggono la stessa identity ignota |
| `K18_N2_envelope_not_base_gap` | NULL | Share=3/4 dentro l’inviluppo N2 della base certa |
| `K19_cardinality_between` | TRUE | BetweenInclusive conserva tutti i body UNKNOWN nella base |

Sono 35 esempi K mirati, 180 casi piccoli a seed fisso con membership/body nei tre stati e 30 casi aggiuntivi con identity finite alternative. La membership della base, i body ternari e la deduplicazione sono verificati separatamente. N2 non viene sostituito da una ricerca di un completamento globale favorevole.

## 6. Share ponderata: prova locale e limite combinatorio

Questa sezione riguarda **V.WeightedShare** (07 §7.3), non un nuovo quantificatore K. La grana è occurrence, part e whole usano la stessa occurrence e lo stesso campo peso, non due riduzioni indipendenti. Il peso è determinato e non negativo; il body/part di questi esperimenti è determinato. Il minimo occurrence, la positività del denominatore e l’eventuale minimo peso sono verificati sullo stesso stato.

Il generatore costruisce stati raggiungibili `(numeroOccurrence,pesoPart,pesoWhole)`. Una occurrence certa è inclusa, una esclusa è omessa, una opzionale dà entrambe le transizioni; `SELECT DISTINCT` fonde stati identici a ogni passo. È una programmazione dinamica locale specifica, distinta dai conteggi K e dai bordi LatestN. Il SQL per le piccole sorgenti note è ottenuto srotolando lo stesso helper per ciascuna identity: **conosce il numero e gli ID delle righe sintetiche in fase di generazione**. Il resto di X1 e del K non ponderato deriva invece selezioni/stati attraverso relazioni SQL. Questa differenza limita la prova ponderata: sostituire `VALUES` con una tabella runtime non basta a renderla una strategia di produzione.

| Caso eseguito | Esito PostgreSQL | Oracolo / motivazione |
|---|---|---|
| `W01_fixed_same_weights` | TRUE | Pesi determinati 3 e 1; quota almeno 3/4 |
| `W02_optional_part100` | TRUE | Opzionale qualificante con lo stesso peso in part/whole |
| `W03_subset_gap_false` | FALSE | Part 4; opzionali non part 2 e 6; share=1/2 impossibile |
| `W04_subset_reachable_unknown` | NULL | Part 8; stessi opzionali; share=1/2 soltanto con entrambi |
| `W05_zero_denominator` | FALSE | Peso totale zero: guardia falsa |
| `W06_zero_weight_optional` | TRUE | Opzionale non part di peso zero non altera la quota |
| `W07_minimum_weight` | FALSE | Il minimo peso whole fallisce |
| `W08_precision` | TRUE | Pesi 9007199254740993.01 e 0.02, confronto esatto |

### 6.1 Controesempio riproducibile ai soli estremi

In `W03` la part certa pesa 4; vi sono due occurrence opzionali, certamente fuori dalla part, con pesi 2 e 6. Minimo uno, peso whole sempre positivo. La richiesta è **share=1/2**.

| Opzionali presenti | Peso part | Peso whole | Quota | Test |
|---|---:|---:|---|---|
| Nessuna | 4 | 4 | 1 | FALSE |
| Peso 2 | 4 | 6 | 2/3 | FALSE |
| Peso 6 | 4 | 10 | 2/5 | FALSE |
| Entrambe | 4 | 12 | 1/3 | FALSE |

Esito corretto ed eseguito: **FALSE**. Gli estremi 1/3 e 1 circondano 1/2, ma nessuno stato realizza quella quota. Un algoritmo che restituisse NULL perché la soglia appartiene al solo hull introdurrebbe un falso UNKNOWN. In `W04` la part certa pesa invece 8: con entrambe le opzionali la quota è 1/2, negli altri stati no; l’esito è NULL. Le soglie usate sono esattamente 0,5, senza richiedere literal razionali non terminanti.

### 6.2 Portata del limite

Il problema generale contiene già questa forma chiusa: part certa di peso positivo **S**, occurrence opzionali non part di pesi interi positivi `w_i`, richiesta share=1/2. L’uguaglianza vale esattamente quando:

```text
S / (S + somma dei pesi opzionali presenti) = 1/2
           ⇔ somma dei pesi opzionali presenti = S
```

Tutti i body sono determinati; tutta l’incertezza è nella base registrata. Servono quindi informazioni sulla raggiungibilità delle somme, non soltanto minimo/massimo. Con pesi potenze di due gli stati di somma sono distinti: la prova con dodici opzionali produce effettivamente **4.096 stati finali**, rispetto a 101 stati per cento opzionali K tutti qualificanti. Non viene affermata l’impossibilità di qualunque algoritmo migliore: bitset su somme scalate, meet-in-the-middle o altri algoritmi locali restano da confrontare. La riduzione mostra perché non possiamo dedurre un costo ragionevole generale dal successo dei piccoli esempi.

Il limite sperimentale di quindici identity opzionali rifiuta l’istanza da sedici con `BudgetExceeded`, prima del SQL. È **un limite del prototipo**, non una proposta di limite della 1.2, un risultato UNKNOWN, né prova che il sedicesimo elemento sia intrattabile. I pesi o la part UNKNOWN e le identity incerte nel ramo ponderato sono esplicitamente non coperti. Un prodotto che debba supportarli richiede un algoritmo ulteriore verificato; non è lecito mantenere il limite sperimentale come modifica implicita del contratto.

## 7. Confronto minimo A / B ridotta

### 7.1 Una sola implementazione

`Composer` raccoglie CTE e compone il SQL finale; non possiede nodi Scan/Join/Reduce, visitor, passaggi di lowering, riscritture, ricerca di piani o optimizer. Le relazioni sono piccoli descrittori di confine, contenenti nome, grana, owner, colonne, tipi logici essenziali e identity dei fatti condivisi. Non possono essere percorse come un secondo AST relazionale.

La relazione degli esiti compatibili ha grana `CompatibleOutcome`, non “una riga per profilo”: solo la chiusura finale produce il valore unico. Il consumer finale controlla questa precondizione; un candidato grezzo non può essere consumato direttamente come una relazione di esiti già derivati. I descrittori non sostituiscono la verifica di correttezza dei loro producer.

La variante **A** usa esattamente gli stessi helper con i controlli disattivati; la variante **B ridotta** attiva i controlli. Per tutti i 1.022 casi il SQL A e B è **identico byte per byte**. La differenza non è una maggiore espressività SQL, ma l’esplicitazione delle precondizioni che A dovrebbe conservare per convenzione.

| Errore introdotto nel test | Convenzioni A | Contratto B controllato |
|---|---|---|
| Grana occurrence passata al consumer di selezioni classificate | Non verificata dal descrittore | `ContractViolation` |
| Relazione di un altro owner | Non verificata | `ContractViolation` |
| Colonna contesto mancante | Non verificata | `ContractViolation` |
| Identità dei fatti condivisi persa | Non verificata | `ContractViolation` |
| Body dichiarato numero invece di valore ternario | Non verificato | `ContractViolation` |
| Candidate grain usata come whole di WeightedShare per occurrence | Non verificata | `ContractViolation` |
| Candidate grain consumata direttamente dalla chiusura degli esiti | Non verificata | `ContractViolation` |
| Assemblaggio Last/Previous senza chiave contesto comune | SQL eseguito: falso NULL nel controesempio | Rifiuto esplicito dell’assemblaggio |

Il controllo della chiave nel pairing è specifico dell’helper, non un parser che dimostri qualunque join scritto in una String. Anche i tipi logici sono dichiarazioni dei producer: il descrittore non inferisce il tipo dal SQL. Un helper che mentisse nei metadata potrebbe ancora produrre un errore o una valutazione sbagliata. Servono quindi oracolo e test delle relazioni, non soltanto `require`.

### 7.2 Contratti effettivamente utili

Le prove giustificano un nucleo piccolo:

1. **Grana e identity pertinente:** occurrence, candidate identity o selezione; chiave di deduplicazione e namespace; nel futuro multiprofilo anche KPID/owner come chiave SQL, non soltanto etichetta.
2. **Owner lessicale e schema dei dati consumati:** contesto, identity, body ternario, membership distinta e numeri esatti. Questi sono contratti interni, non nuovi DTO pubblici.
3. **Fatto condiviso e contesto compatibile:** identity del riferimento risolto, dominio/precisione governati e chiave che vincola tutte le occurrence della stessa alternativa. In produzione il resolver deve preservare anche la revisione semantica corretta; il prototipo non implementa un registry.
4. **Legame part/whole e guardie:** medesimo stato della base e medesimo peso, con i minimi consumati prima della chiusura. Non basta un flag generico “certain/possible”.
5. **Confine di chiusura ternaria:** il consumer sa se sta componendo body K secondo N2 oppure selezioni/stati internamente compatibili. Non è autorizzato a sostituire l’uno con l’altro.

Non occorrono nuove interfacce esportate per ogni punto. In Go possono essere poche struct interne e helper con argomenti specifici, più verifiche ai confini delicati. I test Python non scelgono package, firme pubbliche o API warehouse.

### 7.3 Costo e astrazioni eliminabili

Su 300 compilazioni dello stesso piccolo caso E, A impiega **9.981 ms** e B **10.739 ms** complessivi nel processo Python. È una singola micro-misura, dominata da rumore e carico locale; non consente di stimare un overhead percentuale affidabile. Non vi è costo SQL aggiuntivo nel percorso valido: i testi sono uguali. L’errore evitato è osservabile, mentre il beneficio di un’intera nuova gerarchia di tipi Go non è dimostrato.

Possiamo quindi mantenere una sola implementazione semplice, con i contratti minimi incorporati negli helper. Si possono eliminare: un AST relazionale autonomo per questa prova, una fase di lowering distinta, passaggi generali di riscrittura, optimizer, solver generale, cinque sottosistemi indipendenti e un modello universale di bounds. Non si elimina la logica semantica locale: celle/compatibilità E, quantificazione N2 e stati/pesi della share hanno responsabilità diverse.

Rispetto al rapporto 12, l’esperimento non richiede i suoi eventuali contratti completi di piano fisico, `ReadView`, generazioni, materializzazione o snapshot persistenti. Non modifica quel documento né dimostra superflua ogni sua idea in futuri carichi. Conserva la sola responsabilità di interpretazione condivisa e adattatori di dialetto; la scelta dell’ingresso warehouse rimane aperta.

## 8. Risultati realmente eseguiti

### 8.1 Suite semantica finale

Comandi eseguiti dalla directory `prototype/` isolata:

```sh
python3 -m unittest -v test_contracts
python3 run.py --out /private/tmp/krenalis-x1-x2-15-93okkmyz/release-evidence
```

Il secondo comando ha avuto accesso autorizzato al solo Docker locale e alle directory temporanee. Non sono stati installati pacchetti Python, scaricate immagini o consultati database condivisi.

| Gruppo | Casi PostgreSQL | Confronto |
|---|---:|---|
| E mirati | 54 | 53 con oracolo; 1 Int64 con prova analitica separata |
| E completamente determinati EC | 425 | Tutte le 85 storie del dominio dichiarato × cinque selettori |
| E pseudo-casuali ER | 180 | Oracolo su tutti i completamenti del micro-modello |
| E con due fatti EM | 80 | Oracolo su domini piccoli e dipendenze condivise |
| K mirati | 35 | Oracolo delle basi + N2 sui body |
| K pseudo-casuali KR | 180 | Oracolo delle basi + N2 |
| K con identity alternative KI | 30 | Oracolo delle identity e deduplicazione |
| WeightedShare mirate | 8 | Tutti i sottoinsiemi opzionali piccoli |
| WeightedShare pseudo-casuali WR | 30 | Oracolo a pesi esatti |
| **Totale** | **1.022 PASS, zero divergenze** | **1.021 esaustivi nel proprio dominio; 3.891 visite complessive** |

“Esaustivo” indica copertura completa del piccolo modello/completamenti dichiarati per quel caso, non dell’intera specifica o di tutti i programmi della grammatica. Il seed fisso rende i casi pseudo-casuali ripetibili; non sostituisce una prova universale.

Sono passati **sette metodi unittest**, inclusi confronto byte per byte A/B su tutti i casi, controlli dei confini e rifiuti. Gli **otto input rifiutati** coprono timestamp ignoto, sequenza parzialmente nota nella stessa parità, proprietà qualificante ignota non modellata, flag condiviso non risolto, candidate identity ignota senza dominio, membership correlate senza modello, part ponderata UNKNOWN e limite di stati ponderati. Non sono conteggiati come filtri 1.2 valutati correttamente: sono casi ancora non implementati nel prototipo, con errore esplicito anziché un falso risultato ternario.

Il probe tecnico `SELECT 1/0` termina con errore PostgreSQL, exit code psql 3. Il runner lo mantiene come errore; non lo trasforma in MATCH, NO_MATCH o UNKNOWN. L’esecuzione finale ha exit code zero, nessun caso semantico fallito e cleanup del container riuscito.

Le prime esecuzioni intermedie hanno coperto 476, 480 e 597 casi, prima dell’estensione finale. Non vengono sommate alla suite finale. Non sono emerse discrepanze funzionali fra il generatore corretto e l’oracolo nei casi coperti. La perdita del contesto è una variante errata introdotta deliberatamente per la prova A/B, non una regressione trovata nel codice applicativo.

### 8.2 Crescita osservata, non benchmark di produzione

| Prova | Righe classificate E / stati finali K o V | Esito | Wall time | Byte SQL |
|---|---:|---|---:|---:|
| `BE_latest10_tie_500` | 500 | TRUE | 48.66 ms | 78997 |
| `BE_last_ordered_500` | 500 | TRUE | 53.94 ms | 78861 |
| `BK_optional100` | 101 | TRUE | 41.15 ms | 16007 |
| `BK_optional60_three_classes` | 9261 | TRUE | 43.73 ms | 10593 |
| `BW_subset12` | 4096 | NULL | 61.96 ms | 8832 |

Il batch dei 1.022 casi impiega **0.890 s**. I wall time delle singole prove includono avvio di `docker exec`, `psql`, parsing, query e trasporto del piccolo risultato. Sono singole misure su dati `VALUES` e un profilo; non sono latenze server isolate, p95, test su indici o capacità di listing/count/export. Non è stato misurato il picco di memoria delle relazioni interne.

La prova Latest10 con 500 pari visita un solo stato di bordo, evitando 500! ordini e i sottoinsiemi di dieci elementi. K con cento opzionali tutti TRUE usa 101 stati; con sessanta opzionali distribuiti fra tre classi ne usa 9.261. La crescita ponderata è qualitativamente diversa. Il SQL contiene comunque i dati sintetici: i circa 79 kB delle prove E non sono una stima della dimensione di un futuro SQL su tabelle.

### 8.3 Verifiche non effettuate

Non sono stati eseguiti Snowflake, suite applicative Go, suite differenziali Record, query multiprofilo, listing/count/export, test di rete o errori parziali del warehouse. Nessun risultato storico del rapporto 14 è presentato come nuovo test di questa esecuzione. L’assenza di variabili d’ambiente Snowflake è stata verificata come solo booleano, senza leggerne o riportarne contenuti; non era disponibile una connessione reale per queste prove.

## 9. Limiti residui e controesempi al di fuori del sottoinsieme

| Ambito | Stato | Perché conta / comportamento del prototipo |
|---|---|---|
| E con dominio A/B e numeri su griglia governata | PASS nel sottoinsieme | Le celle mantengono il fatto comune; non enumerano la storia completa |
| Più di due fatti, altri tipi e predicati | Non coperto | Il costo del prodotto di celle cresce; non è approvato un nuovo limite di due fatti |
| Qualificazione con proprietà ignote per-occurrence, alias o vincoli non esprimibili dai fatti scalari | Non coperto | Un NULL locale non prova indipendenza; serve un algoritmo/contratto specifico prima di generare alternative |
| Timestamp ignoti, ordine parziale oltre i gruppi di parità supportati | Non coperto | Non è lecito sostituire ID o primo evento disponibile a un ordine affidabile |
| E body completi, owned members, unità/valute/date, first-following, LastTwoIntervals e scope relativi | Non coperto integralmente | Le prove non implementano tutti i body o tutte le selezioni interne previste da 1.2 |
| K con identity finite alternative | PASS nel sottoinsieme | Contesto prima di dedup; body N2 dopo la base |
| K identity su dominio aperto/multivalore o inventory/contratti completi | Non coperto | Servono classi di identità/alternative appropriate, non un candidato per ogni riga o una sola identity “unknown” |
| K membership correlate di candidati distinti | Non coperto | Due candidati mutuamente esclusivi ed esaustivi, entrambi TRUE quando presenti, provano Some; trattarli come due opzionali indipendenti inventerebbe una base vuota e un falso UNKNOWN |
| Share ponderata piccola, valori determinati | PASS locale | Gli stati preservano pesi e part/whole |
| Share ponderata generale, pesi/part/identity ignoti o sorgente di cardinalità nota solo a runtime | Non coperto | L’algoritmo attuale è limitato e parzialmente srotolato; `W03` confuta la sola aritmetica degli estremi |
| Grane multiprofilo, ownership e isolamento reali | Non provato | L’owner è controllato nei descrittori, ma manca un KPID SQL su popolazioni reali |
| Snowflake e precisione/overflow dei due backend | Non provato | PostgreSQL numeric e i prodotti incrociati non dimostrano i limiti Snowflake |

Il controesempio delle membership correlate è una dipendenza **della base**, non un’inferenza globale vietata fra body K. Un resolver potrebbe riutilizzare il meccanismo di contesti già mostrato in E e nelle identity finite; questa resta un’ipotesi da implementare e confrontare con l’oracolo, non un PASS acquisito. Senza un modello di quella dipendenza l’ingresso sperimentale viene rifiutato.

Analogamente, la correttezza delle celle ordinate non si estende automaticamente a qualunque confronto/operatore Record o business. Non sono state modificate né duplicate le semantiche dei 26 operatori; il futuro compilatore deve conservare quei renderer come foglie con esito ternario, senza confondere JSON raw e NativeCurrency. Nessun risultato del presente prototipo certifica già quell’integrazione.

I limiti indicano insufficienza degli algoritmi provati, non autorizzazione a cambiare le specifiche, rimuovere capacità o dichiarare normativamente invalido un filtro 1.2 valido. `PrototypeUnsupported` e `BudgetExceeded` servono a non presentare un esperimento incompleto come un motore funzionante. Anche i limiti sperimentali di due fatti E, due fatti identity enum K, dieci valori per fatto identity e minimo K fino a cento non sono nuovi vincoli normativi approvati.

## 10. Esperimenti indispensabili e sequenza successiva

Prima di approvare l’architettura esecutiva completa servono almeno:

1. **Mappatura chiusa dei predicati E ammessi.** Classificare per tipo/firma quali incertezze richiedano celle, gruppi di ordine, identity o altre dipendenze, mantenendo il riferimento unico al valore governato. Estendere l’oracolo a timestamp ignoti, alias di valori, domini aperti e body autorizzati; dimostrare che nessuna forma ammessa viene degradata automaticamente a UNKNOWN.
2. **Base K con dipendenze esplicite e identity aperte.** Realizzare il caso dei due candidati mutuamente esclusivi ma esaustivi, collisioni/fresh identity, qualificazione e identity insieme, dedup su più evidenze. Applicare N2 soltanto dopo la valutazione individuale dei body e verificare All/None/share e guardie sulle basi compatibili.
3. **Share ponderata su sorgenti runtime.** Confrontare un algoritmo esatto circoscritto per stati/somme con alternative adatte ai domini dichiarati, compresi Decimal scalati, pesi ignoti e soglie di uguaglianza. Riprodurre `W03/W04`, misurare crescita e overflow. Una riduzione della semantica o un limite di prodotto richiederebbe una decisione separata esplicita, non può essere assunto dal prototipo.
4. **Stesso nucleo su PostgreSQL e Snowflake reali.** Adattare solo le operazioni di dialetto, eseguire lo stesso corpus e verificare precisione, NULL, dedup e piani generati. Le differenze di tipo non devono far migrare la semantica negli adattatori.
5. **Sorgenti reali e popolazioni.** Sostituire le relazioni sintetiche con sorgenti risolte a runtime; introdurre le effettive chiavi KPID/owner/context e controlli di sicurezza sui risultati warehouse. Verificare isolamento, costi di partizionamento, cardinalità intermedie e cancellazione, senza introdurre materializzazione generale come prerequisito.

Solo dopo la revisione di questi risultati: portare il nucleo minimo in Go con test condivisi; collegare foglie Record e resolver tipizzati; integrare un ingresso warehouse circoscritto; verificare membership distinta da listing/count/export e dai vincoli operativi destination. Le API esatte, la forma del risultato e l’eventuale materializzazione restano da scegliere. Questo rapporto non le approva.

I tre lavori infrastrutturali rinviati restano separati. La loro garanzia può essere un prerequisito di rilascio per funzioni dipendenti, ma non è stata usata qui per introdurre UNKNOWN da latenze tecniche né per bloccare la progettazione dei moduli semantici.

## 11. Risposte decisionali richieste

1. **La B ridotta è sufficiente per X1 e X2?** Per i sottoinsiemi implementati sì: i generatori producono risultati corretti senza IR autonomo. Per la sufficienza completa 1.2 la risposta è **non ancora dimostrato**. Restano soprattutto incertezze non modellate, basi K correlate/identity aperte e share ponderate generali. I PASS non approvano preventivamente B.
2. **Quali contratti tipizzati sono realmente necessari?** Grana/identity/namespace, owner e chiavi di contesto, schema con tipi logici essenziali, provenance dei fatti condivisi, membership distinta dal body, stesso stato part/whole e ownership delle guardie. Poche struct/helper interni possono bastare; il test non richiede una nuova gerarchia pubblica.
3. **Quali astrazioni possiamo eliminare?** Per i casi provati: secondo AST relazionale, lowering separato, optimizer/riscritture generali, solver universale, moduli ermetici per ciascuna primitiva e un unico algoritmo di bounds. A e B condividono una sola implementazione; restano i piccoli controlli utili, non due framework.
4. **Ci sono casi 1.2 non ancora implementabili con gli algoritmi provati?** Sì, sono elencati in §9. In particolare il DP ponderato non è già una soluzione runtime scalabile e la quantificazione su identity/membership non rappresentate non ha ancora un algoritmo completo. Non vengono rietichettati come UNKNOWN normativi.
5. **Serve un piano relazionale autonomo?** Non emerge tale necessità da X1/X2. Le difficoltà osservate riguardano compatibilità e algoritmi semantici locali; un IR non ricostruisce fatti persi e non rende economiche tutte le somme raggiungibili. Questa conclusione non prova che ogni futura ottimizzazione possa farne a meno: oggi manca una giustificazione concreta per aggiungerlo.
6. **Quali esperimenti sono indispensabili prima dell’approvazione?** I cinque di §10: copertura di tutte le firme E, basi K correlate e identity aperte, WeightedShare runtime esatta, corpus differenziale su entrambi i database reali e sorgenti/popoli reali con chiavi e costi verificati. Si può continuare in B ridotta come veicolo sperimentale; l’approvazione esecutiva completa resta successiva.

## 12. Conservazione del repository e controlli documentali

È aggiunto soltanto questo rapporto. I 28 file preesistenti della directory filtering, incluso il rapporto 14 non tracciato, sono stati confrontati mediante SHA-256 con la fotografia iniziale e risultano invariati. HEAD e staging sono conservati; non è stato eseguito alcun commit. Nessun codice applicativo, driver, benchmark, documento 06/07/08 approvato o rapporto 12 è modificato. Le fixture JSON raw/NativeCurrency e i 26 operatori rimangono quindi byte per byte quelli della baseline documentale e applicativa.

`git diff --check` è stato eseguito; il nuovo file non tracciato è verificato anche direttamente per whitespace e consistenza dei riferimenti. L’archivio dell’appendice C è stato decodificato e i sei file confrontati byte per byte con i sorgenti eseguiti. I frammenti SQL delle appendici A/B sono confrontati con gli output effettivi, senza correggerli manualmente per la presentazione.

## Appendice A — SQL E realmente generato per il controesempio A/B

Questo testo è l’output completo di `compile_e` per `E01_report14_shared_pair`, inclusi i dati sintetici e il trasporto JSON del test. `ROW_NUMBER() OVER ()` assegna soltanto un ID tecnico al contesto: non ordina le occurrence. `DENSE_RANK` conserva invece i gruppi di parità del loro ordine dichiarato.

```sql
WITH
raw AS (
SELECT * FROM (VALUES
(CAST('e1' AS text), CAST('1' AS bigint), CAST('0' AS bigint), CAST('A' AS text), CAST('90' AS numeric), CAST('90' AS numeric), CAST(TRUE AS boolean)),
(CAST('e2' AS text), CAST('2' AS bigint), CAST('0' AS bigint), CAST('B' AS text), CAST('140' AS numeric), CAST('140' AS numeric), CAST(TRUE AS boolean)),
(CAST('e3' AS text), CAST('3' AS bigint), CAST('0' AS bigint), CAST('A' AS text), CAST('100' AS numeric), CAST('100' AS numeric), CAST(TRUE AS boolean)),
(CAST('e4' AS text), CAST('4' AS bigint), CAST('0' AS bigint), CAST('B' AS text), CAST('150' AS numeric), CAST('150' AS numeric), CAST(TRUE AS boolean))) AS input(id, ts, ord, category, x, score, flag)
),
occ AS (
SELECT *, DENSE_RANK() OVER (ORDER BY ts,ord) AS g FROM raw
),
fact0 AS (
SELECT DISTINCT val FROM (VALUES ('A'),('B')) AS d(val)
),
contexts AS (
SELECT ROW_NUMBER() OVER () AS ctx,fact0.val AS f0 FROM fact0
),
evaluated AS (
SELECT c.ctx,o.id,((o.category = c.f0)) AS qual,((o.score >= 80)) AS body FROM contexts c CROSS JOIN occ o CROSS JOIN LATERAL (SELECT o.flag AS flag) AS v
),
classes AS (
SELECT ctx,id,
MAX(CASE WHEN qual IS TRUE THEN 1 ELSE 0 END) AS can,
MIN(CASE WHEN qual IS TRUE THEN 1 ELSE 0 END) AS must,
CASE WHEN COUNT(CASE WHEN qual IS TRUE THEN 1 END)>0 AND COUNT(CASE WHEN qual IS TRUE THEN 1 END)=COUNT(CASE WHEN qual IS TRUE AND body IS TRUE THEN 1 END) THEN TRUE
WHEN COUNT(CASE WHEN qual IS TRUE THEN 1 END)>0 AND COUNT(CASE WHEN qual IS TRUE THEN 1 END)=COUNT(CASE WHEN qual IS TRUE AND body IS FALSE THEN 1 END) THEN FALSE ELSE NULL END AS body
FROM evaluated GROUP BY ctx,id
),
classified AS (
SELECT c.*,o.g,o.score FROM classes c JOIN occ o ON o.id=c.id
),
totals AS (
SELECT c.ctx,COALESCE(SUM(r.must),0) AS must,COALESCE(SUM(r.can),0) AS can FROM contexts c LEFT JOIN classified r ON r.ctx=c.ctx GROUP BY c.ctx
),
outcomes AS (
SELECT CASE WHEN ((a.score-p.score)*1)>=((1)*10) THEN TRUE WHEN ((a.score-p.score)*1)<((1)*10) THEN FALSE ELSE NULL END AS value FROM classified a JOIN classified p ON a.ctx=p.ctx AND a.id<>p.id WHERE a.can=1 AND p.can=1 AND p.g<=a.g AND NOT EXISTS (SELECT 1 FROM classified z WHERE z.ctx=a.ctx AND z.must=1 AND z.id<>a.id AND z.id<>p.id AND z.g>p.g) UNION ALL SELECT FALSE FROM totals WHERE must<2
)
SELECT json_build_object('value',(SELECT CASE WHEN COUNT(*)=0 THEN NULL WHEN COUNT(*)=COUNT(CASE WHEN value IS TRUE THEN 1 END) THEN TRUE WHEN COUNT(*)=COUNT(CASE WHEN value IS FALSE THEN 1 END) THEN FALSE ELSE NULL END FROM outcomes), 'outcomes',(SELECT COUNT(*) FROM outcomes), 'contexts',(SELECT COUNT(*) FROM contexts), 'classified',(SELECT COUNT(*) FROM classified));
```

Output effettivo: `{"value":true,"outcomes":2,"contexts":2,"classified":8}`. La variante errata è riproducibile con `compile_e(model, checked=False, unsafe_pair=True)`; il runner salva anche il SQL dei witness e i quattro pairing incompatibilmente assemblati. `checked=True` rifiuta quella richiesta con `ContractViolation`.

## Appendice B — SQL K realmente generato: presenza opzionale e share=1

Questo è l’output completo di `compile_k` per `K02_optional_all_qualifying`. I due stati sono `(S=1,D=1)` e `(S=2,D=2)`. Il denominatore non viene preso da una diversa alternativa del numeratore.

```sql
WITH
contexts AS (
SELECT 1 AS ctx
),
seed_raw AS (
SELECT * FROM (VALUES
(CAST('orders' AS text), CAST('O1' AS text), CAST(TRUE AS boolean), CAST(TRUE AS boolean), CAST('1' AS numeric), CAST(NULL AS text)),
(CAST('orders' AS text), CAST('O2' AS text), CAST(NULL AS boolean), CAST(TRUE AS boolean), CAST('1' AS numeric), CAST(NULL AS text))) AS input(namespace, id, membership, body, weight, identity_fact)
),
seed AS (
SELECT c.ctx,r.namespace,(r.id) AS id,r.membership,r.body,r.weight FROM contexts c CROSS JOIN seed_raw r
),
distinct_base AS (
SELECT ctx,namespace,id,
CASE WHEN COUNT(CASE WHEN membership IS TRUE THEN 1 END)>0 THEN TRUE WHEN COUNT(CASE WHEN membership IS NULL THEN 1 END)>0 THEN NULL ELSE FALSE END AS membership,
CASE WHEN COUNT(*)=COUNT(CASE WHEN body IS TRUE THEN 1 END) THEN TRUE WHEN COUNT(*)=COUNT(CASE WHEN body IS FALSE THEN 1 END) THEN FALSE ELSE NULL END AS body,
MIN(weight) AS weight FROM seed GROUP BY ctx,namespace,id
),
candidates AS (
SELECT d.ctx,d.namespace,d.id,d.membership,(d.body) AS body,d.weight FROM distinct_base d
),
counts AS (
SELECT c.ctx,COUNT(CASE WHEN r.id IS NOT NULL AND r.membership IS TRUE AND r.body IS TRUE THEN 1 END) AS required_s,COUNT(CASE WHEN r.id IS NOT NULL AND r.membership IS TRUE AND r.body IS FALSE THEN 1 END) AS required_f,COUNT(CASE WHEN r.id IS NOT NULL AND r.membership IS TRUE AND r.body IS NULL THEN 1 END) AS required_u,COUNT(CASE WHEN r.id IS NOT NULL AND r.membership IS NULL AND r.body IS TRUE THEN 1 END) AS optional_s,COUNT(CASE WHEN r.id IS NOT NULL AND r.membership IS NULL AND r.body IS FALSE THEN 1 END) AS optional_f,COUNT(CASE WHEN r.id IS NOT NULL AND r.membership IS NULL AND r.body IS NULL THEN 1 END) AS optional_u FROM contexts c LEFT JOIN candidates r ON r.ctx=c.ctx GROUP BY c.ctx
),
numbers AS (
SELECT 0::bigint AS n UNION ALL SELECT ROW_NUMBER() OVER () FROM candidates
),
base_states AS (
SELECT ctx,required_s+t.n AS s,required_f+f.n AS f,required_u+u.n AS u
FROM counts CROSS JOIN numbers t CROSS JOIN numbers f CROSS JOIN numbers u
WHERE t.n<=optional_s AND f.n<=optional_f AND u.n<=optional_u
),
bounds AS (
SELECT s AS lo,s+u AS hi,s+f+u AS d FROM base_states
),
outcomes AS (
SELECT CASE WHEN d<1 THEN FALSE ELSE CASE WHEN ((lo)*1)=((d)*1) AND ((hi)*1)=((d)*1) THEN TRUE WHEN ((d)*1)<((lo)*1) OR ((d)*1)>((hi)*1) THEN FALSE ELSE NULL END END AS value FROM bounds
)
SELECT json_build_object('value',(SELECT CASE WHEN COUNT(*)=0 THEN NULL WHEN COUNT(*)=COUNT(CASE WHEN value IS TRUE THEN 1 END) THEN TRUE WHEN COUNT(*)=COUNT(CASE WHEN value IS FALSE THEN 1 END) THEN FALSE ELSE NULL END FROM outcomes), 'outcomes',(SELECT COUNT(*) FROM outcomes), 'candidates',(SELECT COUNT(*) FROM candidates), 'states',(SELECT COUNT(*) FROM base_states));
```

Output effettivo: `{"value":true,"outcomes":2,"candidates":2,"states":2}`. Negli esempi con identity alternative il contatore `candidates` conta slot contesto×identity, non un numero di candidati globali del profilo; il D semantico è calcolato per stato. La distinzione evita di leggere i contatori del test come una nuova API diagnostica di prodotto.

## Appendice C — Sorgenti incorporati e riproduzione indipendente

L’archivio ZIP contiene solo `model.py`, `compiler.py`, `oracle.py`, `fixtures.py`, `test_contracts.py`, `run.py`, senza dipendenze terze. Dimensione ZIP: **19308 byte**; SHA-256: **`fbe836dfd772f4a2aea875eb2c7a26bae90582abec161ae8abf7c667577df144`**. Non contiene transcript inventati: eseguendolo si rigenerano modelli, SQL e nuovi risultati. Le misure temporali non saranno identiche; devono esserlo gli esiti e il numero dei casi, salvo differenze tecniche esplicite dell’ambiente.

Per estrarre in una directory temporanea scelta, senza eseguire automaticamente il contenuto:

```sh
python3 - /percorso/al/15-verifica-fattibilita-architettura-minima-x1-x2.md /private/tmp/krenalis-x1-x2-reproduction <<'PY'
import base64, hashlib, io, pathlib, sys, zipfile
report = pathlib.Path(sys.argv[1]).read_text()
encoded = report.split('<!-- BEGIN PROTOTYPE ZIP -->\n```base64\n', 1)[1].split('\n```', 1)[0]
blob = base64.b64decode(encoded)
assert hashlib.sha256(blob).hexdigest() == 'fbe836dfd772f4a2aea875eb2c7a26bae90582abec161ae8abf7c667577df144'
target = pathlib.Path(sys.argv[2])
target.mkdir(parents=True, exist_ok=False)
with zipfile.ZipFile(io.BytesIO(blob)) as archive:
    assert set(archive.namelist()) == {'model.py','compiler.py','oracle.py','fixtures.py','test_contracts.py','run.py'}
    for name in archive.namelist():
        (target / name).write_bytes(archive.read(name))
print(target)
PY
```

Dopo aver letto i sorgenti, dalla directory estratta:

```sh
python3 -m unittest -v test_contracts
python3 run.py --out /private/tmp/krenalis-x1-x2-reproduction-results
```

Il runner richiede Docker e l’immagine indicata già disponibile localmente (`--pull=never`); non scarica componenti. Per un confronto indipendente, controllare `results.json` e non soltanto l’exit code. I limiti espliciti sono in `unsupported()`; le istanze riproducibili principali sono `E01`, `K01`, `K02–K04`, `K15–K18`, `W03–W04` e `BW_subset12`. Nessun comando opera sul branch o sui database applicativi.

<details>
<summary>Archivio dei sei sorgenti eseguiti, codificato Base64</summary>

<!-- BEGIN PROTOTYPE ZIP -->
```base64
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
```
<!-- END PROTOTYPE ZIP -->

</details>
