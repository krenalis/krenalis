# Krenalis — Fase 4B: analisi conclusiva della complessità algoritmica

**Stato:** rapporto per decisione sulle strategie esecutive; nessuna modifica normativa e nessuna implementazione di produzione.

**Baseline verificata:** `typed-filter-values@c23777219db49fbf6164b88b4e74ca55a5830f64`. Data: 27 ottobre 2026. Specifiche 06/07/08 v1.2 approvate, immutate. Rapporti 12–15 conservati; il 12 resta una proposta non approvata.

## 1. Conclusione decisionale

**La semantica 1.2 comprende richieste con difficoltà combinatoria reale, ma non tutte le share ponderate sono difficili.** Una parte consistente ammette aggregazioni, monotonicità, formule esatte o algoritmi specializzati. La crescita esponenziale degli stati di un particolare DP non dimostra, da sola, che quella specifica istanza richieda lavoro esponenziale.

Il rischio accertato non riguarda soltanto l’uguaglianza: il minimo peso whole, previsto dall’AST, può rendere difficile anche una soglia ordinata. La combinazione “share≥1/2 e peso whole≥2S” può chiedere esattamente che un sottoinsieme di pesi sommi S.

**Raccomandazione: strategia B del mandato, conservando il vincolo di esattezza della A.** Usare percorsi esatti differenti secondo struttura e costo; se non si ottiene un esito dimostrato entro il budget, restituire un errore tecnico, non UNKNOWN. Gli esperimenti non giustificano oggi la revisione normativa C. Non dimostrano neppure che ogni interrogazione valida sia sempre eseguibile entro costi accettabili.

**La scelta architetturale B ridotta rimane appropriata.** La selezione di algoritmi locali e una migliore composizione SQL non richiedono un piano relazionale autonomo o un optimizer generale. Qui “strategia B” indica le alternative esecutive del mandato, mentre “B ridotta” indica l’architettura di riferimento: sono due decisioni distinte.

Risultati nuovi: **62 esecuzioni SQL su PostgreSQL**, di cui **60 completate con esito atteso** e **due timeout tecnici deliberati**; **30 prove locali** su righe effettivamente lette dal database, di cui **25 completate correttamente** e **cinque rifiutate dal budget bitset**. Non sono centinaia di ripetizioni della suite del rapporto 15. **Snowflake non è stato eseguito.**

## 2. Perimetro normativo e modello di analisi

Fonti principali: [06 §10.3](/Users/marco/src/krenalis/filtering/06-specifica-semantica-riconciliata-v1.2.md:402), [06 §§10.4–10.5](/Users/marco/src/krenalis/filtering/06-specifica-semantica-riconciliata-v1.2.md:418), [06 §19.4](/Users/marco/src/krenalis/filtering/06-specifica-semantica-riconciliata-v1.2.md:1049), [07 §4.1.1](/Users/marco/src/krenalis/filtering/07-ast-canonico-riconciliato-v1.2.md:294), [07 §7.3](/Users/marco/src/krenalis/filtering/07-ast-canonico-riconciliato-v1.2.md:777), [07 §7.5](/Users/marco/src/krenalis/filtering/07-ast-canonico-riconciliato-v1.2.md:846), rapporto 15 §6.

`WeightedShare` usa un solo whole scope, una part aggiuntiva sulle stesse occurrence, il medesimo campo peso, minimo occurrence positivo ed eventuale minimo peso whole. Il peso è non negativo e additivo nell’unità dichiarata. `OrderedOrEqualityLiteralTest<Fraction0To1>` comprende **anche `TestLiteralInterval`**, con estremi opzionali, inclusivi o esclusivi: gli intervalli sono effettivamente ammessi, non una capacità ipotetica. Il filtro può avere dimensione sintattica costante mentre cresce la storia interrogata: i limiti AND/OR e N=10 di altre costruzioni non limitano a dieci le occurrence di V.WeightedShare.

Per l’analisi di una base già risolta alla grana corretta, indichiamo con:

```text
b_i ∈ {0,1}       appartenenza della occurrence registrata al whole
p_i ∈ {0,1}       appartenenza alla part, quando la occurrence è nel whole
w_i ≥ 0          peso della stessa occurrence
D = Σ b_i w_i    peso whole
N = Σ b_i p_i w_i
Q = N/D          definita solo con D>0 e guardie soddisfatte
G = minimo occurrence ∧ D>0 ∧ eventuale minimo peso whole
risultato del mondo compatibile = G ∧ test(Q)
```

Questo è un modello matematico dell’evidenza interna, non nuove variabili o formule del linguaggio pubblico. Le alternative devono rispettare fatti condivisi, dominio, identity e deduplicazione; non sono automaticamente indipendenti. Non si aggiungono eventi esterni mai registrati. Le righe con b_i possibile sono occurrence esistenti con qualificazione incerta.

La chiusura della costruzione è MATCH quando tutti gli esiti compatibili sono veri, NO_MATCH quando tutti sono falsi, UNKNOWN negli altri casi, applicando le specifiche regole d’evidenza. **Non si estende questa ricerca ai body K per rafforzare N2:** con base K determinata resta obbligatorio l’inviluppo composizionale dei risultati individuali. La difficoltà dimostrata sotto usa V, body/part determinati e sola incertezza della base; non dipende da una reinterpretazione di N2.

### 2.1 Denominatore e guardie

- **D certamente zero:** NO_MATCH per tutti i test, compresi uguaglianza e diversità. Non si inventa Q=0, non si esegue 0/0.
- **D certamente positivo:** il confronto è trasformabile mediante prodotti incrociati esatti; restano da verificare gli altri minimi.
- **D incerto fra zero e positivo:** i casi con guardia falsa contribuiscono FALSE; i casi validi consumano il test. Se il test è sempre falso, il risultato può essere NO_MATCH; se la quota è sempre 1 quando definita ma D può essere zero, share=1 può essere UNKNOWN.
- Un minimo peso certamente soddisfatto o certamente impossibile è economico. Un minimo **attivo**, che seleziona combinazioni di occurrence, cambia il problema computazionale.

Un peso mancante non diventa zero. Un peso comune può cancellarsi algebricamente fra N e D soltanto se è realmente lo stesso fatto e la positività necessaria è provata. Le proprietà mancanti o JSON raw non vengono reinterpretate come variabili numeriche: si applica prima il loro contratto semantico. I 26 operatori e la distinzione JSON raw/NativeCurrency restano invariati.

## 3. Matrice dei casi ponderati

Classi: **1** aggregazioni/limiti semplici; **2** algoritmo specializzato polinomiale nel modello dichiarato; **3** DP/raggiungibilità o ricerca di stati; **4** famiglia con rischio combinatorio esponenziale, in parte con riduzione formale sotto. Le classi 3 e 4 non implicano che ogni istanza richieda un solver o tempo esponenziale. Le righe coprono combinazioni rappresentative, non il prodotto cartesiano di ogni ipotesi.

| Caso | Ipotesi indispensabili | Test / esito da determinare | Classe e metodo esatto plausibile |
|---|---|---|---|
| M01 | Base, part, pesi e dedup determinati | Tutti gli operatori e intervalli | **1:** SUM esatte, guardie, un confronto; O(n) operazioni aritmetiche |
| M02 | Appartenenze indipendenti, pesi noti, part nota o con scelte locali; tutte le guardie già certe | Una soglia ordinata | **1:** estremi esatti della somma con segno (§4.1), O(n) |
| M03 | Come M02; solo minimo occurrence e positività da verificare, pesi noti | Una soglia ordinata | **2:** DP di minimo/massimo per count saturato al minimo e flag di peso positivo; O(nm), spazio O(m), senza asse delle somme |
| M04 | Minimo peso whole attivo; pesi positivi esatti e presenze opzionali | Anche `>`, `>=`, `<`, `<=` | **3/4:** vincoli congiunti sulle somme; Subset Sum è un sottocaso (§5.3) |
| M05 | Guardie certe, base incerta indipendente, pesi noti | Uguaglianza o diversità interne a [0,1] | **1** per prove decisive; altrimenti **3/4** per raggiungibilità dello zero della somma con segno |
| M06 | Come M05 | Quota uguale a 0/1, quota certamente costante, soglia fuori dai limiti | **1/2:** sfruttare non negatività, cancellazioni ed estremi raggiunti; non serve DP generale |
| M07 | Come M05 | Intervallo con una sola soglia | Come confronto ordinato; attenzione a bordo aperto e guardie |
| M08 | Come M05 | Intervallo con due bordi, anche non degenere | **1** se tutti dentro o certamente disgiunto; altrimenti **3/4**, perché l’intervallo può contenere solo valori non raggiungibili |
| M09 | Base certa, pesi noti, part incerta indipendente | Ordinati / uguaglianza / intervallo | Ordinati **1**; uguaglianza/intervalli possono richiedere **3/4** su somme ponderate della part, anche con D noto |
| M10 | Base e part certe; pesi indipendenti su intervalli continui realmente dichiarati | Confronti e uguaglianza | **1/2:** domini convessi, somme lineari/frazionarie e fattibilità; bordi aperti/infiniti richiedono informazione di raggiungibilità degli estremi. È un’ipotesi matematica, non un default per Decimal |
| M11 | Base e part certe; ogni peso varia su un intervallo intero contiguo della medesima griglia | Uguaglianza | **2:** i totali part/non-part sono due intervalli interi; equazione diofantea con vincoli sui due totali (§4.3) |
| M12 | Pesi mancanti ma con domini discreti/sparsi o passi differenti; oppure optionalità con peso minimo positivo | Uguaglianza/intervalli | **3/4** possibile: il dominio `{0,w_i}` basta a ricreare Subset Sum. “Mancante” da solo non prova durezza |
| M13 | Pesi ignoti senza limiti sufficienti, dominio governato disponibile | Qualsiasi test | Provare prima invarianti/guardie; poi applicare l’evidenza disponibile. Non assumere densità o inventare un intervallo finito per poter decidere |
| M14 | Un fatto condiviso semplice: stesso peso comune positivo, stesso predicato part o pochi valori critici | Test appropriati | **1/2** o poche alternative governate; la dipendenza può semplificare invece di aggravare |
| M15 | Più dipendenze condivise non decomponibili, appartenenze correlate | Tutti | Dipende dalla struttura: **2** se parametrizzabile in pochi fatti; **3/4** possibile in generale. Le prove indipendenti non si trasferiscono automaticamente |
| M16 | Identity certe e normalizzazione determinata | Tutti | Dedup ordinaria prima della share, tipicamente hash/sort; costo separato dal test numerico |
| M17 | Identity/dedup incerte, pesi e part associati agli stessi oggetti | Tutti | Necessari vincoli sulle alternative della base; bound semplici per alcuni casi, ricerca **3/4** possibile per altri. Non è qui dimostrato un teorema di durezza per ogni forma identity 1.2 |

La classe 2 non significa “solver generale”: count DP, gcd, ordinamento, matching o una piccola equazione intera sono algoritmi specifici. Nessun caso viene classificato come costoso soltanto perché contiene un NULL.

## 4. Algoritmi economici e loro condizioni di correttezza

### 4.1 Una soglia ordinata, guardie certe

Per q=a/b, con b>0, D>0 e 0≤a≤b:

```text
Q >= q  ⇔  C = bN-aD = Σ b_i (b p_i-a) w_i >= 0
```

Le altre tre disuguaglianze usano lo stesso C e il rispettivo confronto con zero. Per una occurrence opzionale con part determinata e peso noto, i contributi possibili sono `{0,c_i}`. Con part localmente incerta possono essere `{0,-a w_i,(b-a)w_i}` oppure il sottoinsieme ammesso dalla sua presenza. Se le scelte sono indipendenti, gli estremi sono esatti:

```text
Cmin = contributi certi + Σ minimo dei contributi ammessi per occurrence
Cmax = contributi certi + Σ massimo dei contributi ammessi per occurrence
```

Si decide il test tramite Cmin/Cmax, senza visitare tutte le combinazioni. Qui gli estremi sono **raggiunti**; i bordi stretti sono quindi rispettati esattamente. Con domini aperti o infiniti occorre conservare anche se un infimo/supremo è raggiungibile: non basta confrontare numeri sentinella. Per esempio un supremo uguale a zero ma non raggiunto non prova l’esistenza di C=0.

Per uguaglianza, Cmin=Cmax=0 prova MATCH, zero fuori dall’intervallo prova NO_MATCH. **Zero dentro l’intervallo non prova che sia raggiungibile.** Per diversità si applica lo stesso problema di raggiungibilità, mantenendo le guardie. Con guardia falsa, sia `Q=q` sia `Q<>q` sono falsi: non si introduce un NOT generico.

### 4.2 Minimi di count e positività

Con pesi noti e scelte indipendenti, il minimo occurrence m può essere gestito conservando, per ogni count k saturato a m e per un flag “peso positivo presente”, il minimo e massimo C raggiungibili. Sono O(m) stati per passo, O(nm) tempo; se m>n la guardia fallisce direttamente. Questo DP è polinomiale e non ha la dimensione pseudo-polinomiale del peso totale.

Le alternative che falliscono una guardia danno FALSE; non vanno scartate quando si vuole provare MATCH della costruzione completa. Un’ottimizzazione può cercare prima un mondo valido TRUE e uno FALSE: due witness compatibili bastano a provare UNKNOWN. Se non li trova entro il budget, non può dichiarare UNKNOWN soltanto per tale insuccesso.

### 4.3 Pesi ignoti non necessariamente combinatori

Se base e part sono certe e i pesi di ogni gruppo sono indipendenti e contigui sulla **stessa** griglia intera, i totali possibili sono intervalli interi contigui X per la part e Y per la non-part. La richiesta Q=a/b diventa `(b-a)X-aY=0`, con i limiti dei due intervalli: un problema a due variabili risolvibile con gcd/Euclide esteso e limiti sul parametro della soluzione, non una selezione di n sottoinsiemi. La positività e i minimi restano condizioni da verificare congiuntamente.

Questa semplificazione richiede contiguità e medesima griglia; non vale per insiemi sparsi, pesi che condividono vincoli ulteriori, part ignota che assegna pesi interi alternativamente ai due gruppi, o presenze che introducono `{0} ∪ [L,U]` con L>0. Anche il passaggio da Decimal discreto a intervallo reale continuo può inventare uguaglianze inesistenti. Nessuna di queste ipotesi viene dedotta dalla sola presenza di un valore mancante.

Se tutti i pesi sono `k_i*x` e x è lo stesso fatto positivo, Q cancella x: può restare un problema sui k_i, ma l’incertezza di x non aggiunge automaticamente stati. Se tutta la base è certamente part e D è certamente positivo, Q=1 anche con alcuni pesi ignoti. Se D può essere zero, la guardia impedisce quella conclusione globale. I cinque probe di §7.6 verificano questi confini su domini governati piccoli.

## 5. Difficoltà combinatoria effettiva

### 5.1 W03 e riduzione a Subset Sum

W03: una occurrence certa, nella part, di peso 4; due occurrence registrate opzionali, non-part, di pesi 2 e 6. Minimo occurrence uno, D sempre positivo. Le quote possibili sono:

| Somma opzionale s | N | D | Q |
|---:|---:|---:|---|
| 0 | 4 | 4 | 1 |
| 2 | 4 | 6 | 2/3 |
| 6 | 4 | 10 | 2/5 |
| 8 | 4 | 12 | 1/3 |

Share=1/2 è NO_MATCH; share<>1/2 è MATCH. W04 cambia il peso certo a 8: con entrambe le opzionali Q=1/2, negli altri mondi no; entrambi i confronti sono UNKNOWN.

Per una istanza Subset Sum con interi positivi w_1…w_n e target S>0, costruire una part certa di peso S e occurrence opzionali non-part di pesi w_i. La richiesta, di forma sintattica costante, è Q=1/2:

```text
S / (S + Σ_{i∈J} w_i) = 1/2  ⇔  Σ_{i∈J} w_i = S
```

La base vuota delle sole opzionali fornisce sempre un witness FALSE (Q=1); la occurrence certa resta. Se esiste il sottoinsieme J, il risultato è UNKNOWN; se non esiste, NO_MATCH. Per diversità, l’assenza di un sottoinsieme prova MATCH, altrimenti UNKNOWN. Dunque una procedura che decidesse sempre questi esiti distinguerebbe le istanze sì/no di Subset Sum.

Subset Sum è NP-completo nel modello standard degli interi codificati in binario, con lunghezza numerica variabile (§5.4). Nel sottocaso costruito, riconoscere UNKNOWN equivale alla ricerca sì di Subset Sum; provare NO_MATCH dell’uguaglianza equivale alla sua negazione. Il risultato non è “tutti i filtri hanno costo esponenziale”, né una dimostrazione incondizionata di impossibilità: non è noto un algoritmo polinomiale generale, salvo P=NP. [Jin e Wu, SOSA 2019, introduzione](https://drops.dagstuhl.de/storage/01oasics/oasics-vol069-sosa2019/OASIcs.SOSA.2019.17/OASIcs.SOSA.2019.17.pdf).

La riduzione qui è specifica del filtering ed è derivata in questo rapporto. Non richiede valori condivisi, identity incerte, pesi mancanti, denominatore nullo, guardie di freschezza o eventi ipotetici. Bastano pesi determinati esatti, n non fissato e appartenenze opzionali indipendenti. Valori Decimal esatti contengono già gli interi come sottodominio; non servono epsilon o frazioni non terminanti.

### 5.2 Intervalli e part incerta

Un intervallo ammesso può contenere una soglia non raggiungibile. In W03 `[0,49;0,51]` è NO_MATCH; in W04 è UNKNOWN. L’intervallo non è degenere e i bordi sono Decimal esatti. Un intervallo abbastanza stretto intorno a 1/2 può isolare un unico target di somma sulla griglia intera: anche il caso non degenere conserva la difficoltà nel modello a precisione variabile.

Non basta spezzare il test in due condizioni sorelle con inferenza separata: in W03 “Q≥0,49” e “Q≤0,51” sono entrambe UNKNOWN, mentre il singolo intervallo è certamente falso. Si deve conservare l’evidenza interna del test originale, senza cambiare la composizione AND/OR approvata.

Anche con base certa, pesi noti e D noto, una part incerta può produrre somme sparse del numeratore. La famiglia Partition, con peso totale 2S e ciascuna occurrence possibilmente nella part, chiede con Q=1/2 se un sottoinsieme pesa S. Non è l’incertezza del denominatore a causare necessariamente il rischio. La semantica N2 dei body K rimane una questione distinta; non si usa questo argomento per rafforzarla.

### 5.3 Anche una soglia ordinata con minimo peso

La semplice etichetta “confronto monotono” non basta quando una guardia correla la popolazione ammissibile. Con part certa S, opzionali non-part w_i e richiesta Q≥1/2:

```text
test Q≥1/2         ⇔ somma opzionale s ≤ S
minimo peso D≥2S   ⇔ somma opzionale s ≥ S
insieme           ⇔ s = S
```

W03 con minimo peso 8 dà NO_MATCH; W04 con minimo peso 16 dà UNKNOWN, entrambi eseguiti sul database. Le ipotesi soddisfano il costruttore 1.2, incluso l’eventuale `minimumWholeWeight`.

Lo stesso argomento copre tutte e quattro le forme ordinate, con pesi interi e soglia 1/2:

| Test | Peso certo e ruolo | Ruolo delle opzionali | Minimo peso whole | Successo equivale a |
|---|---|---|---:|---|
| `>=` | Part S | Non-part | 2S | s=S |
| `>` | Part S+1 | Non-part | 2S+1 | s=S, perché s<S+1 è s≤S |
| `<=` | Non-part S | Part | 2S | s=S |
| `<` | Non-part S+1 | Part | 2S+1 | s=S |

Questa è una dimostrazione algebrica; le esecuzioni dedicate nuove coprono la prima riga, non fingono di aver provato su PostgreSQL ogni famiglia teorica. Evitare soltanto uguaglianza e intervalli in una futura ipotetica revisione non eliminerebbe tutto il rischio: il minimo peso è parte del problema.

### 5.4 Complessità teorica, precisione e casi pratici

Una DP classica per somme intere positive costa O(nT) tempo e O(T) spazio, dove T è il target scalato utile: è **pseudopolinomiale**, non polinomiale nel numero di bit di T. Algoritmi più avanzati non rendono irrilevante quel parametro; non sono stati implementati qui. [Bringmann, SODA 2017](https://arxiv.org/abs/1610.04712).

Il meet-in-the-middle divide n in due parti, genera al più circa 2^(n/2) somme per metà e cerca i complementi mediante hash o ordinamento: la verifica non deve ricreare il prodotto fra le metà. Riduce l’esponente ma rimane costoso per n grande. [Horowitz e Sahni, JACM 1974](https://www.cise.ufl.edu/~sahni/papers/computingPartitions.pdf).

La classificazione NP-completa usa n **e** la lunghezza binaria dei valori come parametri dell’input. Se un contratto fissa per sempre un range finito di peso e una scala, non si può trasferire senza precisazioni il teorema al solo n: i domini finiti danno limiti alle somme. Tuttavia un range numerico di decine di cifre può rendere impraticabile il relativo DP. Il limite 76/37 dei payload Record non è assunto qui come dominio universale di tutti i pesi Unified: contano il `NumericDomain` e l’unità effettivamente risolti.

Sulla griglia comune, gcd e divisione esatta riducono il target; se il target non è multiplo del gcd si può confutare immediatamente una uguaglianza. Un bitset usa circa T/g bit, più temporanei, ma scale fini o gcd=1 possono rendere enorme anche un importo economicamente modesto. Non è permesso arrotondare al centesimo per far entrare il problema nel budget se il contratto ammette i decimali ulteriori.

**Potenze di due non sono una prova di istanza difficile.** Il DP può produrre 2^n somme distinte, ma per pesi supercrescenti esiste un greedy esatto dopo averne verificato la proprietà. Il rapporto 15 aveva correttamente misurato una crescita del suo algoritmo; quella crescita non era una lower bound per tutte le strategie. Le misure nuove verificano concretamente questa distinzione.

## 6. Strategie esecutive, limiti ed errori

Una sequenza ragionevole di decisioni locali è:

1. **Risoluzione e precondizioni.** Grana, identity, fatti condivisi, unità/precisione e guardie. Una mappatura strutturalmente assente è diversa da un valore ignoto sul singolo profilo; un filtro invalido resta invalido.
2. **Fast path esatti.** Base determinata; D certamente zero; N=D o N=0 provati con D positivo; quote estreme; test già deciso da limiti sufficienti. Non eliminare pesi mancanti mediante SUM che li ignora.
3. **Monotonicità con prova delle ipotesi.** Contributi indipendenti, guardie certe, oppure count DP. I limiti devono riferirsi alle stesse righe e agli stessi pesi; limiti N/D indipendenti non bastano.
4. **Riduzioni strutturali.** Scala esatta/gcd, domini contigui, gruppi di pesi uguali, fattore condiviso cancellabile, supercrescenza verificata, simmetrie pertinenti. Sono fatti dichiarati o dimostrati, non euristiche che cambiano il significato.
5. **Raggiungibilità quando necessaria.** Bitset su target gestibile, DP sparso con dedup degli stati, meet-in-the-middle con ricerca corretta, o ricerca esatta con pruning giustificato. Minimi e dipendenze possono richiedere dimensioni aggiuntive dello stato.
6. **Controllo risorse.** Stime conservative prima di allocare; budget sul lavoro effettivo; timeout/cancellazione; errore se non è disponibile una conclusione dimostrata. Non trasformare l’esaurimento in UNKNOWN.

Il motore può fermarsi presto quando ha già una prova: per UNKNOWN possono bastare due alternative con esiti opposti, per MATCH/NO_MATCH possono bastare invarianti o limiti decisivi. Non occorre enumerare tutto per principio. D’altra parte una prova di assenza di una somma può essere il caso più oneroso: l’assenza di un witness entro un tempo assegnato non prova che non esista.

I budget degli esperimenti — dieci millisecondi per due probe, otto milioni di bit per il bitset locale, limite di righe del MITM Python — **non sono nuovi limiti normativi** né una proposta dei budget di prodotto. Una condizione valida che li supera resta valida; è l’esecuzione a non essere completata. Una strategia alternativa esatta può completarla.

Per API e UX occorre distinguere “dati decisivi ignoti” da “risultato non disponibile perché l’interrogazione non è terminata”. Listing e count non devono essere presentati come completi assegnando NO_MATCH/UNKNOWN ai profili non valutati. Un export non deve trattare il fallimento come segmento vuoto o successo parziale nascosto. Non sono progettati qui nuovi endpoint, codici errore, formati diagnostici o protocolli di export.

La scelta fra algoritmi di WeightedShare è una decisione semantica/esecutiva locale condivisa. Non è un optimizer relazionale generale. Le eventuali operazioni fisiche dipendenti dal dialetto non devono trasferire ai driver la decisione su che cosa significhi UNKNOWN.

## 7. Esperimenti eseguiti

### 7.1 Ambiente e riuso

Stessa baseline e stessi sorgenti del prototipo 15, copiati senza modificarli in `source15/`. Sono riusati modello, generatore `compile_k` e oracolo per sei casi piccoli: W03/W04, potenze 8/12, interi larghi 12 e Decimal fini 12. Il vecchio limite sperimentale delle quindici opzionali non viene rimosso nel codice originale.

Nuovi artefatti isolati: `/private/tmp/krenalis-filtering-16-mtfrls9x`. Python 3.13.2 su macOS 27.0.1 arm64; Docker 29.9.2, PostgreSQL 18.6 aarch64, immagine locale `postgres:18.6-alpine3.24`, ID `sha256:d7a8005067f55a2f8ea369e767545fd467b1943a4f0f1a48b7dc44f06c366109`. Niente rete, porte pubblicate, volumi, database o credenziali condivisi. Container temporanei da 512 MiB/due CPU; 256 MiB per i cinque probe di domini. Tutti rimossi al termine.

Le righe sono caricate in tabelle sintetiche **del solo container isolato**. Il generatore delle query riceve scenario, tipo/test e scala, non il numero né gli ID o i valori delle occurrence. `experiment_events` è letto a runtime; un `ROW_NUMBER` tecnico ordina l’elaborazione della somma, non il significato temporale degli eventi. La dimensione del SQL rimane sostanzialmente costante mentre aumenta n. I `VALUES` del setup sono soltanto caricamento delle fixture e non il corpo delle query misurate.

### 7.2 Famiglie limitate e oracoli

| Famiglia | Cardinalità opzionali | Pesi / scopo |
|---|---|---|
| W03/W04 | 2 | Controesempi originari, più diversità, intervallo e minimo peso |
| `power` | 8,12,16,20 | Potenze di due: avversarie al DP esplicito, facili per greedy verificato |
| `wide` | 12,16,20 | Interi positivi larghi a seed fisso, target somma della prima metà +1; istanze senza soluzione verificate dall’oracolo, non dichiarate “le più difficili” |
| `cents` | 16,64,256 | Prezzi ripetuti in centesimi 100/250/500/1000, gcd=50, target metà del totale |
| `fine` | 12,16 | Quantità intorno a 100–104 con piccole perturbazioni esatte a scala 12: economicamente vicine, griglia numerica ampia |
| Determinata / zero | 0 / 1 | Quota 3/4 certa; peso totale certamente zero |

Per n≤20 l’oracolo percorre i sottoinsiemi in Gray code, senza usare gli algoritmi di raggiungibilità confrontati; può fermarsi con due witness opposti sufficienti. Nei casi NO_MATCH percorre tutto il piccolo dominio. Per cents64/256 si usano due witness espliciti, base opzionale vuota e prima metà delle righe: bastano a provare UNKNOWN e non vengono descritti come enumerazione di 2^256 mondi. L’oracolo del rapporto 15 conferma inoltre i sei casi di riuso.

Le prove locali Python leggono **i valori effettivamente restituiti dalla sorgente SQL runtime**, con parsing Decimal esatto prima della scalatura intera. Non ricevono numeri passati a float. Il processo separato consente misure di memoria locali; non costituisce una decisione di spostare le query di produzione in Python o fuori dal warehouse.

### 7.3 Significato delle misure

Si usa `EXPLAIN(ANALYZE,BUFFERS,MEMORY,FORMAT JSON,TIMING OFF) CREATE TEMP TABLE ... AS SELECT ...`, con `work_mem=4MB`, JIT disabilitato e normale timeout di cinque secondi. Il risultato di una riga è conservato nella tabella temporanea per verificarlo **senza rieseguire** la query. I tempi server includono tale wrapper e l’instrumentazione; non sono benchmark di produzione.

**`MEMORY` di EXPLAIN descrive il planner, non il picco completo dell’executor.** Vengono raccolti separatamente VmRSS/VmHWM del backend Linux, prima/dopo, e temp block scritti. VmHWM include processo, librerie e pagine condivise residenti; la differenza non è una misura esatta delle sole allocazioni della query. Il root dei buffer include i figli, quindi i blocchi non sono sommati nuovamente. Il totale RAM del database/host non è misurato. [Documentazione PostgreSQL 18, EXPLAIN](https://www.postgresql.org/docs/18/sql-explain.html).

Per i timeout non sono disponibili misure finali: “n/d” non significa memoria zero. Per Python si riportano tempo dell’algoritmo con tracemalloc attivo e picco di allocazioni Python; i log conservano anche ru_maxrss del processo. Tempi e memorie Python non sono direttamente confrontabili con quelli del backend. Preparazione del container e caricamento dati sono esclusi dai tempi server.

### 7.4 Stati, testo SQL e memoria

| Caso / algoritmo SQL | Somme finali | Stati di tutti gli stadi | Byte SQL | Tempo server ms | VmHWM prima → dopo, KiB | Temp block scritti |
|---|---:|---:|---:|---:|---:|---:|
| `power8_dp` | 256 | 511 | 1567 | 1.794 | 14892 → 18928 | 0 |
| `power12_dp` | 4096 | 8191 | 1568 | 9.711 | 14892 → 20336 | 0 |
| `power16_dp` | 65536 | 131071 | 1568 | 120.315 | 14896 → 39236 | 389 |
| `power20_dp_budget` | n/d | n/d | 1568 | timeout impostato 10 ms | n/d | n/d |
| `wide16_dp` | 65536 | 131071 | 1567 | 125.444 | 14908 → 45088 | 464 |
| `wide20_dp_budget` | n/d | n/d | 1567 | timeout impostato 10 ms | n/d | n/d |
| `cents16_dp` | 145 | 939 | 1568 | 2.384 | 14912 → 19068 | 0 |
| `cents64_dp` | 589 | 18423 | 1568 | 16.617 | 14920 → 21364 | 0 |
| `cents256_dp` | 2365 | 301479 | 1569 | 232.357 | 14924 → 41616 | 847 |
| `fine16_dp` | 65536 | 131071 | 1579 | 120.087 | 14984 → 42744 | 464 |
| `power12_legacy15` | 4096 | — | 8832 | 20.358 | 15064 → 25480 | 0 |

Il DP runtime conserva `(stadio,somma)`, non una maschera di tutti i sottoinsiemi. `UNION` deduplica gli stati; somme uguali condividono uno stato. PostgreSQL conserva però anche stati di stadi precedenti per la ricorsione: 301.479 stati complessivi in cents256 a fronte di 2.365 somme finali. Una DP con frontier rotante può ridurre questo costo; non è una necessità della semantica.

Il SQL DP di queste query è circa 1,6 kB; dopo la sostituzione del solo nome scenario ha due forme, per scala 0 e 12. Il vecchio generatore 15 usa circa 8,8 kB per dodici opzionali e conosce le identity alla compilazione. Il runtime nuovo non introduce un piano autonomo: risolve quella limitazione mediante una relazione letta dalla tabella e ricorsione locale. I byte del caricamento dati, naturalmente O(n), non sono nascosti come se la storia non dovesse essere memorizzata.

Esempio di differenza di memoria osservabile: il planner del vecchio power12 alloca 4.096 kB, quello del DP runtime power16 512 kB. Il picco VmHWM del backend aumenta comunque con gli stati dell’esecuzione. Non si usa la riduzione della memoria del planner per dichiarare risolto il costo delle somme raggiungibili.

### 7.5 Meet-in-the-middle: distinguere algoritmo e piano SQL

La prima forma SQL deriva le due metà, ma il piano PostgreSQL verifica ancora il predicato di complemento come filtro su coppie. In wide20 scarta **1.048.576 coppie**, nonostante soltanto 2.048 somme finali fra le due metà. È complessità introdotta dalla rappresentazione/esecuzione, non un costo inevitabile di Subset Sum.

Sono confrontate, nella stessa esecuzione isolata, la forma iniziale, le relazioni terminali con complemento già calcolato e semplice join, e le medesime relazioni con `INTERSECT`. La sola CTE non forza un join efficiente: il piano osservato sceglie ancora un nested loop. `INTERSECT` produce invece `SetOp`, strategia `Hashed`, sugli insiemi terminali.

| Sorgente | Forma iniziale ms | CTE finali + join ms | CTE finali + INTERSECT ms | Confronti scartati iniziali / join |
|---|---:|---:|---:|---:|
| `power20` | 5.733 | 5.140 | 5.480 | 1023 / 0 |
| `wide16` | 14.784 | 6.649 | 2.605 | 65536 / 65536 |
| `wide20` | 192.856 | 63.988 | 4.861 | 1048576 / 1048576 |
| `fine16` | 14.456 | 6.290 | 2.237 | 65536 / 65536 |

Tutte le varianti restituiscono gli stessi risultati. In wide20 l’ultima forma ha 2.037 byte di SQL e circa 20 MiB di VmHWM osservato, senza il prodotto di un milione di coppie. Il picco non viene paragonato come se fosse uno SLO. Le CTE `MATERIALIZED` qui sono relazioni locali della query sperimentale: **non** snapshot o materializzazioni persistenti dei risultati del rapporto 12. Questa è una correzione locale del generatore, non l’introduzione di un optimizer o di un IR.

La forma con `INTERSECT` è provata soltanto su PostgreSQL. Non è certificata come strategia Snowflake. Il costo ideale della ricerca fra due insiemi non può essere attribuito a un SQL che in pratica ne calcola il prodotto: questa verifica fisica resta necessaria in entrambi i driver.

| Sorgente / algoritmo locale | Metrica stati | Tempo algoritmo ms | Picco allocazioni Python, byte | Esito tecnico |
|---|---:|---:|---:|---|
| `power16/greedy` | 16 | 0.0242 | 832 | completo |
| `power20/bitset` | 349526 | 0.1460 | 222208 | completo |
| `wide20/mitm` | 1947 | 4.3964 | 186620 | completo |
| `wide20/bitset` | n/d | 0.0345 | 2552 | budget superato prima dell’allocazione |
| `cents256/bitset` | 1183 | 0.8598 | 3076 | completo |
| `fine16/mitm` | 511 | 1.0819 | 44868 | completo |
| `fine16/bitset` | n/d | 0.0283 | 2360 | budget superato prima dell’allocazione |

La colonna stati del bitset conta bit accesi fino al target; quella MITM somma le cardinalità dei due insiemi troncati al target; quella greedy conta gli elementi esaminati. Non sono la stessa metrica. Il budget bitset è controllato **prima** della grande allocazione: il tempo di un rifiuto non è tempo di soluzione.

Con cents256, gcd e ripetizioni rendono piccolo il bitset. Con i pesi a scala 12 il target scalato è enorme e il bitset viene rifiutato; MITM risolve i piccoli n provati. Con potenze di due il greedy prova la supercrescenza e risolve senza costruire tutti gli stati. Non si deduce da questi successi che MITM sia sempre economico su storie grandi.

### 7.6 Esiti e valori/pesi ignoti

| Prova SQL eseguita | Esito |
|---|---|
| `W03_dp` | NO_MATCH |
| `W04_dp` | UNKNOWN |
| `W03_ne` | MATCH |
| `W04_ne` | UNKNOWN |
| `W03_interval` | NO_MATCH |
| `W04_interval` | UNKNOWN |
| `W03_guarded_ge` | NO_MATCH |
| `W04_guarded_ge` | UNKNOWN |
| `fixed_dp` | MATCH |
| `zero_dp` | NO_MATCH |

L’intervallo dei probe è quello chiuso `[0,49;0,51]`; i minimi peso sono 8 per W03 e 16 per W04. `fixed` verifica quota≥3/4, `zero` uguaglianza a 1/2 con D=0. Bordi esclusivi e varianti ordinate diverse da `>=` sono analizzati algebricamente, non presentati come nuove esecuzioni SQL.

Cinque probe aggiuntivi leggono da tabelle runtime occurrence e domini governati di uno/due fatti. Un piccolo enumeratore SQL di completamenti è confrontato con un oracolo Python separato; è un **oracolo locale** per questi domini finiti, non una strategia di produzione proposta per tutti i valori mancanti.

| Caso eseguito | Dominio / situazione | Mondi dell’oracolo | Esito SQL |
|---|---|---:|---|
| `unknown_weight_anchor` | Peso ignoto x∈{0,1}, tutta part, più peso certo 3; share=1 | 2 | MATCH |
| `unknown_weight_guard` | Solo peso ignoto x∈{0,1}, tutta part; share=1 | 2 | UNKNOWN |
| `unknown_weight_hole` | Part certa 4, peso non-part ignoto in {2,6}; share=1/2 | 2 | NO_MATCH |
| `unknown_part_hole` | D=12 certo; part certa 4, pesi 2 e 6 con part indipendente ignota; share=3/4 | 4 | NO_MATCH |
| `shared_part_fact` | Due pesi 1, medesimo fatto part g∈{0,1}; share=1/2 | 2 | NO_MATCH |

Qui NULL del peso/part rappresenta informazione governata non disponibile, con dominio dichiarato dalla fixture. Non è JSON Null raw né una nuova interpretazione degli operatori Record. Non viene dimostrato che il futuro registry sappia già fornire ogni dominio sparso della fixture: se una fonte fornisse soltanto limiti più larghi, non sarebbe lecito attribuirle questi stessi esiti. Questi test non provano un algoritmo completo per pesi mancanti su domini illimitati, intervalli continui, domini discreti arbitrari o dedup incerta. Mostrano che non è corretto né scartare sempre l’ignoto né trasformarlo sempre in UNKNOWN della costruzione.

### 7.7 Errori tecnici e limiti della verifica

I due DP a venti opzionali hanno volutamente `statement_timeout=10ms`: entrambi terminano con SQLSTATE **57014**, exit code psql 3, senza valore semantico. Le strategie alternative completano le stesse richieste e verificano UNKNOWN per power20 e NO_MATCH per wide20. Il limite è dimostrato come evento tecnico; non è una stima del minimo tempo necessario.

Cinque esecuzioni bitset vengono rifiutate per il target oltre otto milioni di bit: wide12/16/20 e fine12/16. Non vengono allocati quei bitset e non viene restituito UNKNOWN. Le prove MITM corrispondenti completano con NO_MATCH. Questo comportamento sostiene la selezione di strategie esatte, non una policy sugli UNKNOWN.

I totali conclusivi sono: 45 job SQL principali (43 completi, due timeout), cinque probe di incertezza, dodici job del raffinamento fisico (tutti completi); trenta job locali (25 completi, cinque budget superati). **Zero divergenze negli esiti completati.** I timeout e i rifiuti non vengono contati come PASS di membership. Il primo tentativo aveva un errore del runner — nome scenario usato come nome container — corretto e rieseguito; i suoi errori Docker sono esclusi dalle misure conclusive. Il container rimasto da quel tentativo è stato rimosso separatamente.

Non sono stati eseguiti Snowflake reale, carichi multiprofilo, query di produzione, suite Go applicative o benchmark di throughput/concorrenza. I test non validano la futura API warehouse, la sicurezza dei decoder di produzione o tutte le combinazioni dei domini semantici. Nessun risultato della suite storica del rapporto 15 viene aggiunto ai conteggi come nuova esecuzione.

## 8. Rischi analoghi in E/K/T/P

| Famiglia | Casi con algoritmi circoscritti plausibili | Rischio da non mascherare |
|---|---|---|
| **E** | Un numero fissato di fatti scalari confrontati con valori noti permette celle sui valori critici; un dominio infinito non impone di enumerarne ogni valore. Uguaglianza su testo può richiedere classi osservate più “altro”, non ogni String possibile | Con k fatti, il prodotto ingenuo può crescere come O(c^k); vincoli fra fatti, alias e ordini non totali richiedono compatibilità vera. Questa è anche complessità nel parametro k, non prova che ogni E sia NP-hard. Le selezioni congiunte non possono diventare UNKNOWN per evitare il lavoro |
| **K** | Base determinata: N2 è lineare nei body. Optionalità indipendente/dedup nota: conteggi per classi e minimi. Identity completamente libere, senza altri vincoli, ammettono talvolta limiti distinct semplici | Basi correlate e identity aperte con associazioni/dedup vincolate possono richiedere alternative di partizione/assegnamento. Enumerare tutte le partizioni può essere pessimo ma non prova necessità. Non è autorizzata l’inferenza globale vietata fra body K distinti |
| **T** | Con timestamp noti, bucket e giorni distinti sono raggruppamenti. Con scelte indipendenti fra giorni, il massimo distinct può essere matching; se i giorni possibili sono intervalli, il minimo numero di giorni può usare stabbing greedy | Domini discontinui, fatti condivisi e vincoli “giorni distinti per settimana” con assegnamenti correlati possono creare problemi combinatori. La formulazione generale con insiemi arbitrari può contenere hitting set, ma non è dimostrato qui che ogni tale dominio sia ammesso dai contratti 1.2. Non si assume quella capacità per introdurre un solver |
| **P** | Due–cinque passi fissati, una key comune e membership nota delle liste: unnest/intersezione/join sul medesimo valore. Una linearizzazione di ordine parziale non va enumerata tutta per verificare una singola catena ammissibile | Fanout delle liste e tuple O(n^k), con k≤5 fissato, possono essere onerosi ma non sono da soli NP-hard in n. Key ignote, ordine, assenze e fatti condivisi possono richiedere ricerca compatibile ulteriore; non è provata una riduzione generale per il P ammesso |

Per T, il massimo dei giorni assegnabili con scelte indipendenti è il numero di giorni copribili da eventi distinti: matching sul grafo evento→giorno. Il minimo su intervalli è diverso: scegliere punti che colpiscano gli intervalli. Queste osservazioni hanno ipotesi precise e non risolvono automaticamente QPC, streak, MaximumGap o dipendenze comuni. MaximumGap conserva i bordi finiti normativi. Non si estendono i risultati di Subset Sum indiscriminatamente alle cinque primitive.

Il rischio generale è confondere tre cose: rappresentare molte alternative, doverle realmente esplorare e poterle riassumere esattamente. La scelta B ridotta permette tutti e tre gli algoritmi locali appropriati; non garantisce da sola che il riassunto esatto sia sempre piccolo.

## 9. Raccomandazione e condizioni per la decisione finale

| Opzione del mandato | Valutazione |
|---|---|
| **A — Semantica integra, algoritmi esatti e limiti tecnici** | È il vincolo di correttezza da preservare. Non giustifica una promessa che ogni query valida termini rapidamente o che un unico DP basti |
| **B — Stessi esiti, strategie diverse secondo il costo** | **Raccomandata.** I casi realistici ripetitivi, i domini fini e la cattiva esecuzione del join richiedono percorsi differenti. L’errore resta possibile se nessun percorso dimostra l’esito entro le risorse |
| **C — Futuro cambiamento normativo mirato** | Non giustificato oggi. Se i carichi reali mostrassero ostacoli sostanziali, valutare esplicitamente la combinazione costosa di incertezza, peso e guardie, con impatto business. Vietare solo `=` non risolve il caso dei minimi peso. Nessuna restrizione viene applicata alla 1.2 |

Per approvare l’architettura esecutiva completa restano indispensabili misure su distribuzioni reali di n, scale/gcd, ripetizioni, quantità di profili e frequenza di basi incerte; prove sul database Snowflake; piani e picchi sotto concorrenza; corretta cancellazione e trattamento degli errori nel chiamante; verifica dei percorsi per pesi ignoti e dipendenze non coperte. I budget di prodotto devono essere scelti e resi riconoscibili come limiti di esecuzione, senza cambiare la membership.

Adottare A/B comporta accettare esplicitamente la possibilità di errori di risorsa su richieste valide. Se il requisito di prodotto fosse invece una risposta semantica garantita per ogni richiesta entro uno SLO fisso, questi esperimenti non lo dimostrano e la sola scelta di un’architettura semplice non potrebbe garantirlo.

La correzione minima alla progettazione è prevedere nel modulo semantico condiviso **più algoritmi locali esatti con precondizioni esplicite**, non un nuovo linguaggio o un piano relazionale autonomo. Le stime e i controlli di risorse devono rispettare il confine di fiducia del warehouse prima di allocare stati, bitset o payload. Un semplice dispatcher specifico di WeightedShare è distinto da un optimizer generale.

Identity Resolution/pubblicazione Profiles, acquisizione/attribuzione KPID e destination matching rimangono separati e rinviati. I dati degli esperimenti sono coerenti per costruzione. Non si introducono generazioni, watermark o snapshot persistenti per risolvere un problema aritmetico/combinatorio.

## 10. Artefatti, riproduzione e conservazione

Sorgenti nuovi isolati:

| File | Righe | SHA-256 |
|---|---:|---|
| [cases16.py](/private/tmp/krenalis-filtering-16-mtfrls9x/cases16.py) | 43 | `74f3f3937540b3758a354368bbfb18b1af7e9d71b9c4947ed1742e0418008800` |
| [local16.py](/private/tmp/krenalis-filtering-16-mtfrls9x/local16.py) | 85 | `0d69988b6a077322a97bff176dc24eedce3a2372c17d1a5fbe10c7636460b792` |
| [run16.py](/private/tmp/krenalis-filtering-16-mtfrls9x/run16.py) | 229 | `20ee629d8260ef36886be72f8c84a65fe50286414e4915ca0fd139f94cc74b7b` |
| [sql16.py](/private/tmp/krenalis-filtering-16-mtfrls9x/sql16.py) | 108 | `807b68b187ef7f334a74ac3c16cf74f7474554f54d703c24fd0086b6b33a5ac3` |
| [uncertainty16.py](/private/tmp/krenalis-filtering-16-mtfrls9x/uncertainty16.py) | 108 | `ae6763104b80085708ef3848a0f25ca49da72c04b1a8efd4b51a2c8130305409` |
| [refinement16.py](/private/tmp/krenalis-filtering-16-mtfrls9x/refinement16.py) | 81 | `cd68ebecf07e56133fa1cce046f647a76296f0c83b838f345ad2d977c5b19b13` |

Ingressi verificabili: [runtime_dp](/private/tmp/krenalis-filtering-16-mtfrls9x/sql16.py:63), [runtime_mitm](/private/tmp/krenalis-filtering-16-mtfrls9x/sql16.py:79), [runtime_ordered](/private/tmp/krenalis-filtering-16-mtfrls9x/sql16.py:100), [solve](/private/tmp/krenalis-filtering-16-mtfrls9x/local16.py:9), [oracle](/private/tmp/krenalis-filtering-16-mtfrls9x/uncertainty16.py:38), [refined](/private/tmp/krenalis-filtering-16-mtfrls9x/refinement16.py:9), [intersected](/private/tmp/krenalis-filtering-16-mtfrls9x/refinement16.py:26). Il generatore/refinement è soltanto sperimentale PostgreSQL. Il raffinamento effettua una sostituzione locale verificata sulla forma nota dell’helper originale: non è un parser SQL o un meccanismo generale di riscritture da portare automaticamente in produzione.

Output conservati: [risultati SQL](/private/tmp/krenalis-filtering-16-mtfrls9x/verified/sql-results.json), [risultati locali](/private/tmp/krenalis-filtering-16-mtfrls9x/verified/local-results.json), [ambiente e riepilogo](/private/tmp/krenalis-filtering-16-mtfrls9x/verified/summary.json), [probe pesi/part](/private/tmp/krenalis-filtering-16-mtfrls9x/uncertainty-evidence/summary.json), [raffinamento e piani](/private/tmp/krenalis-filtering-16-mtfrls9x/refinement-final/results.json). I file `.stdout` contengono JSON EXPLAIN e probe di memoria; i `.stderr` conservano i timeout. I SQL generati sono nelle stesse directory di evidenza. Le copie originali `source15/` mantengono gli hash del rapporto 15.

Comandi conclusivi, eseguiti soltanto nell’ambiente isolato:

```sh
python3 run16.py --out /private/tmp/krenalis-filtering-16-mtfrls9x/verified
python3 uncertainty16.py --out /private/tmp/krenalis-filtering-16-mtfrls9x/uncertainty-evidence
python3 refinement16.py --out /private/tmp/krenalis-filtering-16-mtfrls9x/refinement-final
```

I 29 file preesistenti in `filtering`, inclusi 06/07/08 e rapporti 12/13/14/15, sono confrontati con la fotografia SHA-256 iniziale e conservati. Nessun codice applicativo, driver o benchmark è cambiato. HEAD/staging e file estranei sono preservati; l’unica aggiunta nel repository è il rapporto 16. Nessun commit. Sono verificati whitespace, riferimenti ai casi e identità dei sorgenti incorporati: dodici file fra sorgenti nuovi e storici estratti byte per byte e 62 SQL rigenerati identici. Le quattro riduzioni ordinate di §5.3 sono inoltre controllate algebricamente su S=4/8 e tutti i quattro sottoinsiemi della fixture: 32 verifiche, senza contarle come esecuzioni SQL.

### Appendice A — Esempio SQL runtime realmente eseguito

`wide16_dp`: il generatore non conosce i sedici valori; `experiment_events` li fornisce a runtime. La query contiene anche misure degli stati, utili al solo test. Il `SUM` qui opera su pesi **determinati** del sottoinsieme sperimentale; non viene proposto per scartare pesi ignoti.

```sql
WITH RECURSIVE
src AS MATERIALIZED (
 SELECT id,membership,part,weight*1::numeric AS w
 FROM experiment_events WHERE scenario='wide16'
), items AS MATERIALIZED (
 SELECT ROW_NUMBER() OVER (ORDER BY id)::bigint AS i,w FROM src WHERE membership IS NULL
), totals AS (
 SELECT COALESCE(SUM(CASE WHEN membership IS TRUE AND part IS TRUE THEN w ELSE 0 END),0) AS a,
 COALESCE(SUM(CASE WHEN membership IS TRUE AND part IS FALSE THEN w ELSE 0 END),0) AS b,
 COUNT(CASE WHEN membership IS TRUE THEN 1 END) AS certain_count,
 (SELECT COUNT(*) FROM items) AS n FROM src
), fold(i,g) AS (
 SELECT 0::bigint,0::numeric
 UNION ALL SELECT f.i+1,gcd(f.g,t.w) FROM fold f JOIN items t ON t.i=f.i+1
), meta AS (
 SELECT t.*,GREATEST((SELECT g FROM fold ORDER BY i DESC LIMIT 1),1) AS divisor FROM totals t
), normalized AS MATERIALIZED (
 SELECT i,w/m.divisor AS w FROM items CROSS JOIN meta m
), reachable(i,s) AS (
 SELECT 0::bigint,0::numeric
 UNION
 SELECT r.i+1,r.s+CASE WHEN v.take=1 THEN t.w ELSE 0 END
 FROM reachable r JOIN normalized t ON t.i=r.i+1 CROSS JOIN (VALUES (0),(1)) AS v(take)
), outcomes AS (
 SELECT ((a+b+r.s*divisor)>0 AND certain_count>=1 AND ((a)*2 = (a+b+r.s*divisor)*1)) AS value FROM reachable r CROSS JOIN meta m WHERE r.i=m.n
)
SELECT json_build_object('value',CASE WHEN COUNT(*)=COUNT(CASE WHEN value IS TRUE THEN 1 END) THEN TRUE WHEN COUNT(*)=COUNT(CASE WHEN value IS FALSE THEN 1 END) THEN FALSE ELSE NULL END,'final_states',COUNT(*),
'all_stage_states',(SELECT COUNT(*) FROM reachable),'divisor',(SELECT divisor FROM meta)) AS result FROM outcomes;
```

Nella variante MITM con `INTERSECT`, le relazioni terminali hanno il seguente contratto fisico locale; il resto delle due ricorsioni rimane quello del generatore. Questo frammento è estratto dal SQL eseguito, non una nuova API:

```sql
left_final AS MATERIALIZED (
 SELECT (m.a-m.b)/m.divisor-l.s AS complement FROM left_s l CROSS JOIN meta m WHERE l.i=m.n/2
), right_final AS MATERIALIZED (
 SELECT r.s FROM right_s r CROSS JOIN meta m WHERE r.i=m.n
), hit AS (
 SELECT EXISTS(SELECT complement FROM left_final INTERSECT SELECT s FROM right_final) AS found
)
```

### Appendice B — Sorgenti incorporati

L’archivio dei **sei nuovi sorgenti** ha 13469 byte e SHA-256 `e4ec2e5b896ba1861c1d94f07e803986212341b6cec7cb6dd18c08bb9a0552c3`. I sei sorgenti storici del prototipo 15 sono già incorporati nel rapporto 15: si riutilizza quell’archivio immutato, SHA-256 `fbe836dfd772f4a2aea875eb2c7a26bae90582abec161ae8abf7c667577df144`, estraendolo in `source15/`. Non occorrono pacchetti Python terzi. La riproduzione richiede Docker e l’immagine locale indicata; `--pull=never` impedisce download automatici.

Per estrarre senza avviare gli esperimenti, passando i due rapporti e una directory temporanea nuova:

```sh
python3 - /percorso/16-analisi-complessita-algoritmica-unified-fase-4b.md /percorso/15-verifica-fattibilita-architettura-minima-x1-x2.md /private/tmp/krenalis16-reproduction <<'PY'
import base64, hashlib, io, pathlib, sys, zipfile
target = pathlib.Path(sys.argv[3])
target.mkdir(parents=True, exist_ok=False)
def unpack(report, marker, digest, destination, names):
    text = pathlib.Path(report).read_text()
    encoded = text.split(marker + '\n```base64\n', 1)[1].split('\n```', 1)[0]
    blob = base64.b64decode(encoded)
    assert hashlib.sha256(blob).hexdigest() == digest
    destination.mkdir(exist_ok=True)
    with zipfile.ZipFile(io.BytesIO(blob)) as archive:
        assert set(archive.namelist()) == set(names)
        for name in names:
            (destination / name).write_bytes(archive.read(name))
unpack(sys.argv[1], '<!-- BEGIN COMPLEXITY16 ZIP -->', 'e4ec2e5b896ba1861c1d94f07e803986212341b6cec7cb6dd18c08bb9a0552c3', target,
       ['cases16.py','local16.py','run16.py','sql16.py','uncertainty16.py','refinement16.py'])
unpack(sys.argv[2], '<!-- BEGIN PROTOTYPE ZIP -->',
       'fbe836dfd772f4a2aea875eb2c7a26bae90582abec161ae8abf7c667577df144', target / 'source15',
       ['model.py','compiler.py','oracle.py','fixtures.py','test_contracts.py','run.py'])
print(target)
PY
```

Dalla directory estratta, dopo la lettura dei sorgenti, eseguire i tre comandi di §10 con nuove directory output. I tempi e VmHWM non saranno identici. I due timeout a dieci millisecondi sono probe deliberati e il numero effettivo di timeout dipende anche dall’ambiente: controllare gli errori tecnici, non convertirli in esiti. Le cinque prove con domini finiti non vanno generalizzate a un solver completo dell’incertezza.

<details>
<summary>Archivio dei sei nuovi sorgenti, Base64</summary>

<!-- BEGIN COMPLEXITY16 ZIP -->
```base64
UEsDBBQAAAAIAAAAIQASK91T/gIAADoHAAAKAAAAY2FzZXMxNi5weZ1VbW/TMBD+nl9hKk12WpMm
WVeNSpNAAn4AIPgwTZVJLsVS4kS2uxam/XfOL206toFGPiS27+255+6cyWTyjjSwI6YSrfjeAoG9
qOzrHcjND0sa0clWguFE9ZYIolBz2H5vZUUa2VrQpOl1J2w2mUySRvcdqYUVVSuMAUNkN/TajkdR
AyrZifYgfR+2QdZoDC57dbT9GA+SuNdC1X2XJMnbo1eGlr9AXX3RW0gTf0Q+V6CElv0qIfgo0cGK
GKv9zuf0c9yHVM2K2O3QwjUec5Jl2Y0XVqCtkGo9CG1HE0cWepTKkiuSP1Ds7Q/QXhNFNKeJl9bQ
OG3YgDbMQNukAVnAU9neaRf5dOpkmXd/lN+KdotkXpHrAxlsn06jFdJP9uiaeK/ZKVxOHhx5YJyE
CDHn9OYYRTZEqJ/sNqtB9Z1Uwnl/haB8iFsXIgA5Qe4eLaQB8tWJPmjda0ZhP0BloQ6dRDZa1tGU
kw120aHG2AI+0TpWgKZHxxrsVqtQEIa8sdv0EQzsAceqiZU2LAKrhAlsHXqA0W/5OeV0J60CYyhn
tMTtkqacLvBFThUXzyhe0siVg6E84Ze8KHmx5GV+QsmxWAE7tgErp1MZ4Etnhx28AabSMVuPOBPD
AKpmRzANHfod6Dt1jxjMdgAtVaVBGKk2lEdCnX8XQL0u0vn8PI1eR5BPQNRqg/jCIGWf/IcVyzzP
Zyp9Lg+0yZxFAI8Bz99wfC/OQ2LrlyW2kzWEvNxqfRgMykfS9unY2rHgPlmz7VjYX6/UfF7epLPi
iayXfLng5cXy+cqwIs9RI+cX+MV1nl7Ls8XN/9QJV9aEfDQMILD31/7sZQlhActnCvj3NGby7CKd
utujKGeu2/6WhJuZeAMjpj9mOYxdQ+/283nwd5/d7c/CcpUXZX1P/0lHI1Wsboyz9mNelAc6xvCP
KTnInqwzDlzMJAz57MGUN3IPtY+K/yW8wdyG4eji7D+4A69ogRM92v0C3aOZ+6xPLj83/jnl6CA/
zH7kxwdPfgNQSwMEFAAAAAgAAAAhANULVoUNBQAAjQwAAAoAAABsb2NhbDE2LnB5lVZtb9s2EP7u
X0FkWCUlsmO3+1CkYYANS4sCKzC03b4ErkBLZ5mzRGokFdto9993fJFDJU6BGrAlk8fn7p7neOTZ
2dntnpWGNLJkDWGNASWY4fegc6J6QbggawV6QzolS9AaNFlLReRKg7pnqwZIC61Uh9nZ2dmEt51U
hvyjpchJy8wGMUDLXpWQE31ASMNbfDWKldCyBp1OJpOyYVqTj8Hwt76qwdzuS4AKqtS+dIZLkV1N
CH46tMU1FayJls09pC2YjaxysgNeb4x1wRQCBPOaUBfIrC6r9DzYZG6Kr3GWiSosID+T2q+xHwWm
V4K8ZY2GfJEnuNwyINdJBLtP63zhwQIyDt/tLi9rx9HOkhcmls4qeLq8pKQeYvDxE0pJUivM+ZA8
RCFVBQpwEnNVBtkYJeDi7IXgwoYzP44dfYflD3jB5Y5c02HleNJBMq7hOTkS2HegUERhiA8Xqfq3
5yiz5eceBNG9tRClAqZtaBrnQZSQZCNXQ+QXlOwi2lvGBUaNCXmynmal4B6UxlhCetkz+Q1QJzI8
epmOnTvNoxiQ1Zw0II6858nj7EpQhq95yQwkJyRdcaPBRJLitE/sYkFuyOtiPp/b7zjI70vgMYdi
AjeuSSTMyi0gck1e54htv8QuihRomd4ix+ni+jod4smy6eJoYO3R4GGgA7ali6dyBGpOi+ChnyoQ
0FP3/Oaf19e7LCMvXGhPFjjvdsfZl9yaz/CnKGUvTJpljzVcSdmkDvXmJvSDF4ssd4sTPzAtG951
uLk8nW+INqih1U7xknBNLMeOtROytty0Y1HjKrl59UN6fnj/+UMsX8MRnryaE+kaH7ZlJXcj8XhV
NUBjl5eXL4/TG4Z9UdO75XHESMOawiWo6bhR3LOmx72LQg5Yd1cef5kPA/7/1fLRTtN9q+nX+X+j
wWNdeOCn0ttV5BslX/cXO2e9t9ZuFGnEwWvqBRrj+qRmDCUTVWrNx+0kTtH2FEvO2CqUBhOHUPFT
59kD3y2WD8GEofkyy2NYrJydJEPZILgtkVAeXuG/bc63SknlG2VpnGHXue4dCgiF9OcX7Des1/as
HajPR2fXT+SdYodpKSsgIPoWFLP1kJNSCowIK2UoWXsAvyG2aVl2bA3h3v/9T3e62eqaObyyVwrn
QgF02Ee57Id6sMlvXXtlogbbF+LyipSvMSa6/ZJub24WWbwDtmOty40Fqqi1/zI4mzxqAjRYuc2M
DmuzSeMuFIXtDopQkGi9dOc3YmPHCCAE8LAm09gojm/AoafbUiiPz6qHfHsR9b2BJ+ts8vRuMCYK
pUVXRSFYC0XhekVR2POkKJLhAnNoJKuovSTN7FuKF6OZNhUXns4VoBZAh4vTDCNVvWYoynHo41+f
fn13W3y6/eNtNlM9Otgr7cmNrlYIyhR2Rzfu3qm9fs2w0ax95wQVZo06XEVtoRdVHmq+AsN4Q/1d
K8R+l4RKXubHkUBAPORZTpbxFtR9Y2jFS5NafGQ1KWXbNYCnZ+780tg7HQfhHx4O3L3wmZ5KmD0P
y6vvOh7YLMBu2CR3D/SoUlwajhS/7i5hTS0VN5u2aDFFmp7gceoYzs4XeNqO1nYH5EoU9uwpnC7F
6oBJIUysFUZfuP9V4fdzmmFPGgGhxMVm1xa+QI4o/u95urBFbmupa5jBoZbSpGJqx0Xid8Zi/vKX
7CQiW2MKR8Afr7wfct7h9cmkrvyrvu106qNBxv8HUEsDBBQAAAAIAAAAIQBN2/pTxhIAAMo0AAAI
AAAAcnVuMTYucHmlWntz2ziS/z+fguutOpAxzEieZG5XCVPlTJRJbvzIyMpkcxotixIhiTFF0gQV
W+fSd7/uBsCXKCdzl6rIJNhoAI1+/LqBo6OjYbYSa5EHsRWnc/gNgyKYBVJwK98kRbQWFrzHQnJL
3AfzwgriZZpHxWotrSAJrbUI5CYXoSW+RaFI5sI9Ojp6ssjTNXGax4GUQlrROktz6CzDaF6oz4sc
2EVpUn58pxt0bzGP1jAf/fGtelXfsqBYxdHMfPsIr0/MCPkyC3Kc/iqQSMStrzJNuLUGIm5lcVAs
0nzNLbmZZXk6FxJWJrfwg2tV7OewfNn/2bCXc5EEeZRK9VXextW33zci33LrdpMWIixF5odZ9byO
inX1luahAHEpViTxipm4XwUbWUTfxJMnT0KxsObpeg1CtmFRMMEoyTaFd5kmQk02hbefe87giQX/
clFs8qS2KhdGVB1VP/rlhbgvvHG+EXweZNADJrQp8DO1Ga76r2OmsRLzG/Fj0/Aac9aDGnZIES2s
3FWznaehUN2oaxBJYY2UnIZ5nub2gon7TMxBtEYUsMC52rNligKLCu+hzm43sOBdFiDkfDI47fV6
0x1z6hKirzAbvTZUDl8WuQjWNgpHr+NbEG+E9CZTegNdBN65h8Tuf11fXb5VDbbifLeKYjAT6A28
8yiznWpVJHD6FOtv5Scagyee5u7mwZ2vntVMmoTSDbJMgGDpzdkfYJIMpvWFql56maAUM2HHwUzE
enKyCIqN9I6ypQ+LD/0FrMFmz1B7nkkRL54pAuYc1Zkujq6H58Nfxkpus00Uh346+wp7ZDMag3H2
QMPs4CmX0r+JZoyDWuLqk6X9oNjuLDIA9sd6dH09mEwGMgvmYjCdHtuT3sk/p8cOcwaDWbSMkoKz
1d36u3zef774Dh/n5ZGWhkDZBIXwb9F8bWPgYMfkF6QouLWI7kVIaq4FBopLbVYkLWyu9jjgM/70
TkTLVSE9w8yFEcVS5FLvuIhls4tH3JR+ecHx7FiNbMYKX3k9K82tW3cdJdF6s/bVCDh8ArqPUyAP
bNymHfJ+7+nTW1eCUxHOq7K9zaGmnXpX3wUwOWqMxaLwAmBSrHIhV2kcuuDVU2AQFGmuNAF5eGGD
JtlgCDEUMPtbdwPamtcnuzcqjvXaI3a0EByWetWHfEUjqebmKJrLA/PYgKatWHH26rVq+Jtp0O+v
zavu8LrsoN5fmVf9/ZX6vpvcummmBp5qBVqAQAt05hDgRKU/t1q0P6QQIKVYJLYmc16f9ioJ/d26
AIltrUWwjuIthjKQELi7LI7mUQHv8eIEg7R1FxUJucMs3kirWAlLrLNiqz7OU3CmIj/RRG7JHia1
WZuRJ4P6NJ49O51WzgWDN0ZAzwtOZrRJlSw8DySv20pN8LxS7fr8dI/RIcO75T0HdYUUkXgeppRE
ifGqrVAUk1hxl2K0AIwQAW4xEgIMYidpPcY6jPpDKABq8PVge3pn5ps8F0nhZTmAmhScZE/FhEjC
pof6DVCEdQPBEOJWshR2/9WruhhrNrbMg61382/75vXrfiUQ2P2bioZGXSGj0EP6f5uhmxRqXsee
2TndxZ1FhQ+jL4uV7Zz0p8gcufyH/k6+xzr5Tq9yKCMRNwhD+/A+6NlwG7TdcWrRSonp2OuXTaUg
cVZ1EaDIzHCO5502JVLf1sXhfQ0WoOTWgx53R0SxIGjJ6p6iXFeWQhjmCC6MLli/wsQsoR0M9Bwc
4qfsfx1EiYnyBDdzzwBP9yxfApuk+EjtWqUUEUrUD/R3m52cwJQYz8XtJgJMSBhMkSN48nQf+oO9
jONAwIWQlyCWWwIreHDXN2GU2wCLZOGnNzWGKOVnDKArcw4TIWAv3ZUZTfc0rS4GfmByl6MDRNxh
EyoKN+tM2hMF8G2wUbQPifaBXKeABEM0qVPHgDG5iYsSYBEQNi/zNClAviBT3HkFVjaA/vKt97Br
UvhJsBYeuwFFDOJInqitAlS4Pen/fMKOASzY4HZthJQu/thGUyGZWELPLJXFEiYz6P/D/fkkiDPg
+pN7+lwpTpFvK42spmXw8ISFKTzkiHQ2CfyenIQCcp4VPeZr+pNt4thLxDciOzlJBOhxfgPPCSyO
mtZineZbeHzRP1V95hkAL85OVQ9YIuPNJUM79v14dT3+dTS89t/Dg3/2afzevxiO31+99YocFJtx
WiUQz4EY5FOIxAcVzMluPDZ13DYmVbtdDvb4btNWR6FXknPajuZU664hBKAZwxdKMNx1Cg45TaK5
7Rw/75VEGk23KF6Zvk0XEVVJUm03xL2Y10TGGYDcSCLMRTGffIIfs/EohCp98ABzYfTp2OE9nvMA
BKwBc/8ZzqImUK9SLdacMf6bwVRuGq20XBkLkdluLWw2QeOBLKlKkmiFCE4gb/6ohr/+/VzlSoCX
82KTmZSN1XcF8gNwDTb870jpWt74oLjZSdSUOXobaP3XWXG7J3V4/wY/V5f+cDS6GvnX46uPXp9N
Oc6hkS+S+cPQmcd+GQ3PxkPr0+X51a+/Dt9a47M350NERSKP0KP6YGVJIcsgRbmRdXk1ti4/nZ/z
KLRUIlA1genNAJGtosyapWksgoSDuy3MS0WosTcFiGhetX8cfbg4G32xfht+qWJjFDrOyz8TVm1j
AlpRpZMGPpTusSloTW3yvcWR/aDqC7Z0yaJ2vMfHo09D9fMg3TlgK5C6j3PfDQZ6ls6R07aVEpxV
fVIAjbmzr6M/MIu+msC7s/PrxjSI5eF5aM7WsWdNutg+RMenO07i1azv6sxIdBG/Q+GZiA0Y3DXQ
a9pUm2OPfbi8Ho7G1ofL8dW+tlh/nJ1/Gl5b7Jhx2DP3awqhXU/ROWYvrbPLs/Mv/92hZy+rDc49
ZUA4YAPjdZc6DhiyKVzUFF/FvQkTybcoTxMcnU29B7YCQ2IDU81yzQPAGpZtixU47dpHavAhAknc
eqdmhgOaNtM5fUnxkjm6SmL8GTfWPuhwjbofRSsckRzjw4N7LXL45P6hPu92Nf/IGYUmPwo7OdJH
/JtIdG1VIGvw/xC2eMokvVvEwY3w0SVtQKvYgFIKzlSQ9eNoDah3Hc3YAKItOHGMtAN4wKDsAxEb
sOcXb9iu4R2xlEZFDS4xAQEXZVwuVymgcVn+Wnover2266zQ1zPF55i5hMUakRVaQN9AB5vmQgWe
oz//BOWy/hiO3lxdfxh/wb2apVL8mVwPx5aZvEdzf2lh29eo8NLFQr1gsUSQ6hr3zo4IHFXzdo6P
1hKHPzpWxSI2EyBswcAI/kyG//p4fvbh0rK1NfA3n969G46u+cXw4mr0hb+7Gl2cjS2sjfHxh4sP
l79aV+/eOZZ22ePhxUftrk3B2FcQ0DpD04Olu7naRvYSR3yJCyOlzNM7v0h9hB124VjvRlcXezwK
lJqZNqUDLRlS8FOoA4x44evM2G5SaStuFt9UGA4yCQDd7uBwQrydp33Y9v1Nr7abzKm146YW6Tze
EVxCV8eGpzCOFTAy+5qCeit9LYE7Gyj/ypmqKeC7egIk6qcZRoUgZlQLqHwpR6X1Z9sCfQV9go0S
5M0ASauvchWcvviZDXS93VXvTUp3Je7DaClkQWa6CjLxWL9cgPOaC7sZGfgRTNos58g5xP0uiGPQ
ZzbQu8ZZpeRsUD3zEjcBqTFnpuyZDdTfXTuAHvbnmoC9+M9e/zmjqlekq82wUfu03y95A3Sr5Z5Y
CywPW/QqKBTqcqtCebXid/9Frfi9pybuJgspbqoyMCvEfAVAO4h9gfNgHLMonzBxfc2cvnqtUbiy
Q1UtbVmO7JCT8iwcNjnhdzkijJCT2Xr1cny3dVTZI2RVYAoRWqFm4qqat13POR5dtRavYGYF6g+A
QIn2rt4mjNiyKdadSk2RFNzQ4eMyJr3phA0p6OBejWF/AMzilyRKlg2qj7rREOnIpGTiadHoRiUU
+q0xo2+GobsUhV0yZfxh53Dw9ZmP7gITvhkk1zet8ZnuNgZC67MitN4QIeO9DuklsPcN/FoPj2Bx
NzZSON1qDmYRSQjlRZCAUSMhx9TxALUxpI+QI1kXtFbrk0Q8gAaFvdEU2DWemF1jtR++gg2bj4eZ
luswmPbhZoDvk5tpVc2z2SWOMN5mlGDvzYHvDdxooX4OVffMhHbOwRnhqN8MXam8AyXQb939RNwh
TsztHxFnY5zBIe7UjmpywHgmDLtr9QMUSrLs8oFUZ1U9lCGx6WP+74xqw2A22gMelR7wwTztujwd
GeeulVzowpLZYpqG0w3GTgzto0UOTdQuYH0nh7v1flf1UhV0pT6UaYcT9dnzGJ0FdVQKuvnQGQYv
k7mf+POWybbgqdc6r+C3B2aiLCBL70R+2gPNvotCAQ8dqoWQWPU5Zn6Y+bNNCB6FEUCuTsHtW2cP
Kvd7PxIiWvx/gPGhJelIbGsZc/Y/Ik/BRrHO00Q7r7z6GUznTPAwvzkXbPnubP5unc2LDQCTrbUQ
xXxV3q2Q6SYHtwEgV5I6qTsY7ZsWL60ZwM0QLEJuE0iui2iue7qtnPJOQdhHj2kBrjJuawK19qcP
eHSotWunQPZ+pvz5/XA0LK9DeO3EHTLlt1YUej0EYUqkjP9ydXY+vP5lSCblB8ul3Tni1ejtcGS9
+QL9AeN/OB8PR7Yarlai+XBNRRdgP5mywQA5Os7/cbIv94sjIL06uuvM0IGkC3lnwTZOg1BdEMAn
aUjB2XBVwY8SvIRh3hZAVXj6VsveVIJka9//Dcnte1VKvycd1uNM1CZO+dOywQh86nQZK63kD4wv
7ZIhHUri7R/aiLBUTMVPuVpzUweoFBJqztccdE7a092fXBM/QOIEPoNW2VrYX7Tl/RUvCC0XqzRU
9LOokOieGBmwc2zZNlvmQmBB2FEDqGwIfLHcoC4lcwChEiGVOj6zncPoRo0EXYl76VnKo92feodj
M1ZMo6R2mNnMSMtyq9xKV1U18DoWx+SdDoF8urnh+xC/wFdQyd1m+mKRm20hcZzyds1ez1f94Wb/
9F+ut0X96YLSXbnQj9WzOkB83WQOAn4zosbj+mrK1Ksh+MMSLvJNsaLcpM5jgR4VMD1tbnUB4sDI
xMPcZjDGMzgAYCrrUu68gjEauhA3vKqk1Hr37EFtRVe2Vh6PGUijDtiMX9PAQCtvmdNXKb1X10Te
3HkzL68MXo00qLb1s606atMzHsgm+Nk1MFGFJD73fgKb+9x73rZR6WmOE/xpugXivFmAifNbYrMn
E5vh0ZlCRrT88k4CXv5wHN7RY7kJcoig/vJQT4RUzYsyHhpZR8k8o4IPIMHOgdCj5uAnG8NUFyRK
fs//CVAIYiXdbamaX/Spuc27q0L/10CewTIaybBjLeO/AKyam7sHEzmbY+jFqg4HD52IvnpAV/0X
9t9g3u8gXso06uhZm3L9/kkbhV0ldEcmFPM4wDqKhl8yjuAXv2gFeLYO7vEvFuDrNw1yobBZE3b5
3OdP78D4Wjd8quwphOjYvklBl154RyvdzXGcaSei95AZ5PG4ev3oqba+9mXloXnHpuvbp/4ikC2s
rr80tp8pHAoJUS7WgRUU6nLRs8Umji3jXyy8alTzW3+3RmIjlTQBvi7pMHINISLmVM2CYJVThFTK
ZW0Sc0uF7F7EynNKgEWxuolbu7cEMRDv/oKg0ePava4wCNYJWggZHu1t/wWrZ2x4WZAmY+7c/sav
AQXwMXpkMNgApLGIRM7HQhb8F7xukotm93IRmoN+92+aZHp5mki91WkO+EmuzOof5qF/qu2LHsio
Tv+KMUlYnfRsXKSB/sa5Nf0Zd46LDYRSRUoXJ5za+ahHEnI/Xf52efX5ks/ScKub1BmdZnrnfO94
rmkatBfeb9rWca61TXCv35+NhrQVtt4Kd/g7b1q4HhmUOp2rK0lzqi3nonaz5THX8v/KnNMZ1QFD
TucBntlmm9bVJI3FMphv/dKQH5T48Ah1QLeva+LUp1YNmQ/QsHcTM2Jzm/X9utYY9YLlIZeguoCZ
kDsolVmvwG0cdi1YadFqpf0XAGJo6TvyA7LhCMamplzWq+mIxkJvAhZsBVZZdbYWQRRv0L1C9oYe
dwVfb5L0LgENXqNCzPHIC7GPu386iscQX0EwgAgR7eiSjdNNWCLGKd2CvO+Gk1UW8zi3Vt1cdnJt
F9d/lPk6AjdYzFeiZKvKtrqqVrsy+QMMCUTWxUQNBwn3xaRYs+n3BPY4X+BCjnlPXp3sm9Q/MIgm
x0MlktpDpgBr65Qpc+nSO51stU6QaIwMx5DgucEXdsYXdxmnM5s9NUnWrnWFiub3YxVGIm3XF+kS
T1y7fkY3nfTlmu5r1Q05VJewwNrSb1iD7bq3Jos0o3N7cy2Is59qt3jqF5rKxtZK9ZCPr1ETtVeZ
5Zj8P0735Aks3af45vugEr6P3sP3dcanrmI++V9QSwMEFAAAAAgAAAAhAK00zGjcBgAA/hMAAAgA
AABzcWwxNi5web1YWXPbNhB+56/YKg8kbYSWnIfMuFGmis006shSoyNp+6KhRUhGwkMGQNuJqv/e
xUGKOmI7aaeesUUt9sbut0s3Go1RGiUJyC9LGsOCZpRHMucC8lvKgReZZCmF0fse8PxOBPB7LuSC
U0Wg90vK8TSTkGfJl6DRaDhznqcQRzKaJZEQVABLlzmXG5LhmPNoJlmeVedvLcFxnF8qXg95v9Ks
PeYF9R1NgvcF5V/OHMAfMaNZxFl+BkJyS4kSegYMPWpDU5PypQlIMyHVbbuaLq85Fdd5Ep9VtvG0
fPRa5NTXfMUSFdR4/oZ+nlFkVR+aI2UZS4t0ekfZ4loaO9tcjhPTOdwUuaSxdxslGI2JgFNZ8Awa
buNYkwNOl0k0ox5SSMN1G/4xPll5kRccj26sLJtDlktowqs23AQ6cvX44qU51uojJih8UJpDznPu
uerKZuiFyRQ0g+DFS9ev+zJ3XVfwGXRGcNkZh8Nup9f9K7wAz4FR2AvPx8BiktL0inJxzZZkGXFJ
TOhHq1bz6Mj6sj47y4oU60OrunPg7XBwWSuZKb3FvwI+vguHYXWV7ZXNktJiSP7a8QkwSVPxgFPD
wcdpf3L5Jhx6Pgw+hEPwBsML/HjzJzrsn51dsYUqC9TAyJ1xRoVpzG/ige4I+pNeT9mUuYwSbXRj
53zQ6YWj89AbTS69884oVBr6OwrGw0kInf4FqORUhLFivIOwh0JNCPsXPmn6SntEnB/U+7ajlH1T
8ZVWPOmPH9ao5VtaUEnNKJcRy6azHJsfNXhV6ErTkW+Spy9E82dVNlXS5thSHiMLfztvzfIGSLMq
DAcm/e6gD51er2SbB+y4RRaz2JsHCyKDO2tNaYU5/Dbo9m0pSEBJGbC2FlGWUyqjbaMyOCK/DkOs
mNHYK8NY1DRuSgQuMPvQ6152x9DySUu7H7NbJnJuBGw1SGUqyzmCJvuKjfRAn5C7kzQodagmqGUO
zoeD0cgEpB1PHR87z7a6pEJ6N2hHJUqhF4GYZjkijfpiAWBJYlCtXyFZsGGvU2uSWi6hc0m4algU
n7veqhJb+0ereO0SRaxJKfJy7ZaocxOUmKoBiGXgIagS99Vr9Uf9qm/qy+u26z8JjWZJLtRHnmJl
Y7qyCrYrcBJFUvqrAljDauMHftHxrP2akxq4gQntpILhmid72l63rQbdXjspsbrqiVy/au+lqGSr
ydbc2Z4Sh/2yPPU5tCvoH3k1kPUrUbRhGetuwk9taG30P3IL9B6NVk4Ye65/MGs7wWP6Suub6HUq
V0aqTEU1ZPY0NDX7Fva8brd2dZj2mOHiQDNRCG97jrobmCvBqr2Lf3rOHoQ+/aypT9RQQ9+6CkPW
aKwmiTopHbfr1DReVmP8GUwydoMKPSGjBSWiSH1cISKEgJ+xRCCNxGdB1PKFcwwnJpMMtyq82lnE
Y8xewuQX1YNqIZP0XgYGGziN2QyV4HVZMHEj7Mjo+OqYB+LIotL+6P/YHb+DYXg+GY66H0JnVS0d
a/SBRrPr6CqhiO/i6fhe8XAN7mj+uJbMQEafKV60Th0Cfm2M2Z2hMgvc4GUNfaspoHXXQdX70OlN
QnSx6ROv5Wt3bz1lzFcInhcS0YbuTPdVlbe15tc3vefEHnTbLQJ9aGMDOL5j1X1CJJteFSyJp/nV
J2wyz9UqXbKqlfCauHN1kVNz6y4pK484Lm7mU10X1eHhaVy55xO3vNuKdWuMKY9NNmw/a2qZjs0I
Kks1ZTKt75wPoL+vyrI2eRT6bC3U5vzbSGjOHwLug+Bl30FwdOMWgtL4PiGA3hS6N05YVj7iCIfW
ySmBZS6wjW4pzNk9FpFeYXHiKD+jBM1mz/WCpV5qbIdgRiIDfq5ub3cnFW39YgE0QfdchSHu9zSW
GkFT8cSu+pdNZe7bWHy0oZwDtf60Htu0xCtsiZNT1XN6wh6MExnqEVaF+p/Eas3+b8GqUK+Z3I4w
/KM7Go/Kjmxt3UJywLB+3niOflp00RMRU9D20iB6ngZX/mbJfJ4E5QtVYrhPTrWfcxyn8VNwaZNX
LWKSqqYYFmKeJzTKTHpXtiHWKs2IU9sAdhijHoh21+fjb6BclY9HIPh7oNO49YjJH4RVLIN9RM15
TLl6xX0YVPc26W+g42Og+kT0tF7BPBISEVFeYyA3BcNoyqUMFgWuHMIi4jMYZCCv0ZBBMhAJm1EC
6n9JFZSqfxlBxGmFqQS54b1yL6YzHFqCZQsVLm49ZnGJ5PQr5fn+2mLN4nka3R/YarxDL+h3+k24
9r762MIDNVz+nm7xVtbxtW/fIIyj6721sj8Y73FXtJrEgS1yr89OyYEqP62Vqn3Yq0tVraow/wFQ
SwMEFAAAAAgAAAAhADf6dFBrCAAAaBQAABAAAAB1bmNlcnRhaW50eTE2LnB5jVhtU+M4Ev6eX6Hi
i+RB8QAzN7WVlKcuC56FvUzC5WV27zjK5cRK0OLYRrYJLMN/325JtuMAO0cVidXd6n661epu5+Dg
4Iu8F2QlE1kIsk7vhUpE1I3STSgTspVFIvJc5H0iknIjVFjINCEyJ2FCUhUuY8FJkhYkU2lULjUz
S2O5fHQPDg46K5VuSBQWIBeiFiI3WaqKhmQkVqAIt9b8L5Zg2ABMFWka12xrzHCzsLiJ5aLiXcKy
Y59Dtc5ClQPEmzBHIU7+yNOEk0JuhNmtyuT4U7V3eSOWtyLiZJluNmESGZH8Lm5E7sq0MBJJLpK8
zDudzj9rdxhs+FMk3kyVwuloEpmk216HwN9WyPVN0SN5och3MkoTocmAEIgL8G+XaoSDFcRhdwfx
2hvfFPgRqkuVLoTBlYQboVXolUq3eY8UZRaLK4DOXde91gy0VHPMJ+zhzRNKXjfyxY0S+U0aR73m
OJEuHjKxhBi2fe50IrEiyxCShDkGlxJFqRJypRf4pzEzWia3SbpNAhuiMFnepIpyBmgZ/UC5dpPj
CjXrJacP1HE4Y/DNGT2inB4jweEVNHbs2I0/MLcuQxVZa239/IcGUP6HBiBkovLm4wtvvoRxvu/O
CVj7tG+NnzhG+G2DOoHeNIdqtck6zzyaUcv79JJ3ZyFlrQAwoO+sdxB+4B/fRJjfhEpEQa29Anj8
0uy6gvQ6T0Nav3km+1G6toloShvLbCreh3Epcu/K3oRUQalI5VIQKJG2GLF3zNZMZOOd4naNMq6+
PY5VV18nL5Kw80+ZMYY7/m4rNxYdp9ZQpEUYe+iuVzt01LBRlzIa9J2uGbq8NFuUaxKPyBVpnnNd
1XVFERAeA/eq4uv4XjstlY8QIOUiHKPJPL2lpz6la8+DY2lp0o4detsWEXSChZ6uers8czRumGUi
iZje+vkIulOkJd+bIHmZW5cjZ7e4mN1XR9eoP4xj9uB5DQ1D+IAhNCTHuKCzLBYJs1SbMtAm6nwJ
+WLXpGtbZ6r4LjUSSbqRCdLrQuyZBsMyF1ctrCtK6W8Xs3My8U/nk+nFN7+jo0kGU/J1MPMnF4Ph
xX/9M8I6ZOoP/dMZOfNHUz+YDEb/Yg4Zf/MnhI0nZ/D183/0STi93kKuZVKgDsmRxLVX5Mtk/JWU
gXGR/HbuT3ySL0USKpl6T4jtueNwsk1VHOVM8oVMIpmsIUagqQFwVBng9Onp+Zn2etiAF2Q+uhiP
yGA4rCW3rjw85lu3UvT9u5YMFqWMoyBd/AFNg630ZeArV+NyDEqDgWzJr+OLkUkwsiKgfuVKT6tF
pBD0NL4XURtfY48rV0b8dDwY+tNTv74VnDUi3c+fWxcAoqcPVi6113ZDh+wowRx8oaLO/UaBd6xV
IKez59XpZDydGt/KAG8y3GpzHsrdOxEyGJ1hID1m3fs6+J1JGyZTRjAUebnJ22GogzCdf2XGD40n
0oTTwdRHkyONj8zwydYJfwisI+KPzrR8YkzVsf5lMp5fYrJVBtB8WhYwX4k9CBHcWoSfvHtaPHvR
u6fwGQV2shFhd5yOlcfsaCcH1bKUP9WzGXOeOTWBpHC089GMvXNsOCoUcKnsBcaKW80eYeZVw6M7
UGs4pKS4xJViTj/M3DCKgtDSGe12QRvlStyVEvqWGbT6wM89kNVaUBrwaOUg7OGYypDmwsrpw4e7
uY2kYuJB5kWQ3tppDeXBnwKwCeVh6elDdMu4qLtRoR6b2t6I2lmWXdEohQcYkCjMuvDZ7UYCBsMb
/ag2+isr49hLxL0W63YTUUDUbuE5AYOatBGbVD3C48k/Ppk9y6zMcY0LlLkcT2e/TPxpcA4PwWA+
Ow+++rPz8ZlXqDKH8NAszYs1oO8d/+R+6oZxBkA/uCcfUcMSPmLwXCQBxFbpdw6PXjsuTJbQHttt
LcCarMJkLdjJ0ZHT228Udnzf9V08CDBRhwfArAOZKxFG6FR3vgMPrZqSu0wj4Xmmm7wSzxc6lyG6
+R4mguX74/eIYscD6HO1BZh9wfRtu+vBO4mbx0JkzD1u/MWW0wNnJXSeSZmglK9UqhitBmlyadRO
/z0k2iEOr3GF1odp2ajCJM+wTeV7IbMN5k0XaVe2YwdKkPr7oLh7ET1Y38PHeBT4k8l4Ekxn40vv
mF7zvEGSg8HMo6cTH/oWmQ1+Hvq2uLGqpJFCPBRcRgSbh602tljqEU+/PYgw4TsF2eypy6teOn2y
Z8b0tD1DzXZTcarC3qet8MWyYPdOr4oYHc2HQ4opd4+jTjPmMDqbzP2Gg5fZcOiXAdTMhqOnT8Oy
rf/eaWd7htneejPa5Uquh7zq7RxmVj3u7UnWQT/0VgcXMBRMZuRiNBtXLeXbYDj3p+ypPX088ycJ
/+h01ficeo1hdva4pqu1RCzJ6R+8gP76pPsSOMqaUwEpI/9SaNdD2vbQDjFv+mgJ1VKLP++evLk0
qPz/PZnqbnLTfbz6faIlhb+JQCbXFxnSIXOrVUvyrhTq0TMjZovB4Iq/t74cUhcvpuO4WyULEWA6
M73zkPb/l1Bn79JjG/Gwj7pxGsIUp908mPozkheQSdjbAltFYEbPaf/gUGtzXvXC6LuyXfganakd
wwJa8W0/hrnfPL0Cqh7n9esRBtgzHvJKo1cH2Lphvjh4ECweC+gdOKBbtAbuCibtON7plQpaGgwp
Xl3Y6ius3+9e1MK8SDPd+arKyvHXhnr3bqWvidoYDC6bEI6v9ia3cPNmd1Chsd88T0u1FAG8CUPH
9ezvV65ZMj09BMFKxiIIsFuFkXGbweHfiIdIrkVeMOu5zhFqUbh44LSVIjoFonKTQVE0QhzGNTh9
78Rx+pmCCvz3MjBAQTkLAjyiIIBGFwR4SYMAGp2Zqjp/AVBLAwQUAAAACAAAACEAVo+zOyEIAABf
EwAADwAAAHJlZmluZW1lbnQxNi5webVYWXPbOBJ+96/A+gVkGaKPmaS27OKD4tCxsjo8krybjNfF
okhIQsTLAGjZm8p/324Aumw5s1NboyqTINDo40Nf8OHh4aDkpJ4/K5EmOZF8Kkpe8FKTakr0nJPR
b13S64x7ZM7zmktGykqTBAjzRIuqhD1VrUUh/sNlcHh4eDCVVUHqRM9zMSGiqCupyQ18HrhxImd1
IhVnZJ4oJGLkm6pKRoAJt7vTRHF1+n61W6W8TKSolF1VD/lm7beGy2f20FSaZ0w2JTKJC6ELSwsz
G9p0ztMFkKVVUSRlxlBsrLTkScFqWU04A9uF5nElkzTnBwcHGZ86RDLvwT8/IPCrpJgJMDvclgar
djHPQkrpXGjSHhHvgIyibnQ5JtGXzmg88tzXKbkaDnok51MdK5KTy+FgNCKfB50+KbhOSGHHIGiO
BJIM4CMQYRGUpN3/CGMVekWQtIpg4h8XQSYehapkKw8U+dd1NIxIbqmPz3zUY1o1ZXbgg2JGyZIv
UUkjfoqmIE2vPY6GnXa383v0cUvxt8TADoCxzq2r/JE5L5Q68Jkz7o/Eg6WW+QaLt7g7gJD3n4Df
KpBvA26nHOgqzIONofvQFNO1SwAl+IQHTuD/LTy17oI/mQjFSVspLjFkIikr6VH+VPMU3Ha9neQV
hqCNM6LmSQ1BMsNoK4koUQuIuEnOyYyXXCa6ktR6neS6keVGDcnrPEk5KsLgsH3nyaLUHAIPZe7x
5rWbbxz5L0Nvyw/3ytjrWpZzpw+OMkIiR7vjIIbmpZz//4QAt8JI/1aJ8i84mSIRpedOJKnDVYoM
2nLWIAo3+CU9/yKpgyTL4sTNe7TVqhpNmeQPjZA8C8ey4UAmZyoEWsMFqZXnjrXRISZjD+cC+PIv
4BEUi0xIjz8JpeNqYZkYepOJwzsFcIJHEjR+lYw9H4FVQZkUHBc8uhQZP31PmRmcncCgrpZcmhE6
F6z59w4W1eQaGN9fpFWpwXouw35VcrOq5fPmZDbrLn17dzSrYCCBK6RgeLZaGSSCdG6GsjCvusnz
sOSPhqzVKrleVnIB4xKkmKmCF5V8huG70zO7J60bBYMz/ECam8Fo/GkYjeJrGMTt2/F13IvG14OP
oZaN0sY+pWdgzPnp34P3rSSvQdFfgrNfkUMKjxwA5WUMRwZEiCS99wOoOKJ254E/hDZGBGVSzrh3
dnLib8x3/utK1rbt/ImDiDU8oMwsFgqKWYZGtW631EOp1hPTKuNheAJ+m5E9eL7imSZo5jFUx/T4
9Bi12LIgDDcSzicgebGjNhbHQOWc115wujGX54qfu4Ab2hL6MtxuLFfsPIw9NtaQEp19wwpDp4Z2
wIO/F5i5wHvTRtoSu+ABC5z90tYPr+CD70d4DPpxNBwOhvFoPLgJT+k9Q7lrsQpE1iG9HEZQy8ht
vzv49AmK2bj9oRsRtE0KjNkYnLLUylsFEiSXJ81ERiaYIzQDv5xAmp6LmkyqKudJySCM9fpjyTHT
kRISgBQpuxl2eu3hV/KP6OuaJXDz/Qu642EmeE04vwCqWkIcTqn33TZRno1o/wc7YePhbWQf31WQ
QmoEsGJU5sf5uZPv0/ujvbu/i6PTH6x/2+2yq3Z3BCyW27uMToItUStuZhPNYbe1Trk8sQPtUUg7
/VE0HGMNGLxGlPyz3b2NRoQewVkFmKk9NM4/oltQWHcx7Gi73+5+/X3P2VxQ/3/C7iE07aczmqlA
Qf3m/g7NyqmZbSrDnRbTU+xhlxzF5cmE52xamqzq0VXxoDv9rc88uipLMRqLRcCUcFxZl3rKtqq+
/yJIDLQPeTgtvRdq4M+DYDt2th3RmB4ZvY5ogKHi+8FSoiHovRh/gPK/S/qai4asplLIFzo02B+O
ojHBXByDo4f0194HeoFT34QOq+nUjJUGZzDn4WI+pO8UvTg8Ml26RyccYOIUTjb6ctNtd/qeO0r2
4fbqCtoD1ot6g+FXdjUYQmNJPo8GfTbu9Dr9T2RwdeUTF6PjqHfj4rPgiWqggMa2NGEHQY+sWatW
tFrGuorx0uBBK2Oajpe79AVd6ZhMAXaA6U/AqjOT3naQ3cD3mpOFgUFHUSq2lEldg5sZueHW1ean
LKzeUChBFMiUnmMTPCZ5w6HG7zEgMV2S23pHDSW9J0Ktff3VFvRPrPavFjCDL5N84ZVQmPZ4pyt/
QsF+nZTQOSEhy0Sq36B2O+g1XC7JZQU9IIYR7oIujNDP2MBdiRwPZ3sB38GMQ0fVx5nxcw3+BQVu
xPWgpm+LWlsXIG5Qob8vzpHX3eLexPLCtkYbpoyONOa6GVbpS1vWYdROdQP95RASFnxtlGc7GjOK
BGQIrcsj1MnJM9lddVy6VVUravqzxcrIH/6bNqCejyu69cmfm3N53L+N53tOBbudn5zKjpg3uZt5
49JveesaavQCD9Ny6BLwIxS/pNShTaDOt+2LrXwzfJmQ7QtLeTx51tCj5bw0DQUDN4f+MS5UaPS5
O7m/oxF0Dw1eEcgYUhO0ALhUinK2S3bjZldUttmMbcSGLnDdpI1Y82Q2UMxzK/JMyqAr6zG26U6S
wJkga4paeY4Isn4G+TM8c1zMvSjfaquldaFw3QJh/GIHjsGwp2tSuqpNk7zqwhj9ZauB2m4K15NG
mGrAxeVzaA7rWzWx+Do1/Q2HeKWRezNVNTLlMVy2zt69D93/igL76Zk7TAzXvZzHMTa3SWZPDzJW
MOdPmZhxpT1nvcXPafJz/BzRFn4XtYQSuofGx9sbhEEco/PFMWSLOMabXBzTc3ujO/gvUEsBAhQD
FAAAAAgAAAAhABIr3VP+AgAAOgcAAAoAAAAAAAAAAAAAAIABAAAAAGNhc2VzMTYucHlQSwECFAMU
AAAACAAAACEA1QtWhQ0FAACNDAAACgAAAAAAAAAAAAAAgAEmAwAAbG9jYWwxNi5weVBLAQIUAxQA
AAAIAAAAIQBN2/pTxhIAAMo0AAAIAAAAAAAAAAAAAACAAVsIAABydW4xNi5weVBLAQIUAxQAAAAI
AAAAIQCtNMxo3AYAAP4TAAAIAAAAAAAAAAAAAACAAUcbAABzcWwxNi5weVBLAQIUAxQAAAAIAAAA
IQA3+nRQawgAAGgUAAAQAAAAAAAAAAAAAACAAUkiAAB1bmNlcnRhaW50eTE2LnB5UEsBAhQDFAAA
AAgAAAAhAFaPszshCAAAXxMAAA8AAAAAAAAAAAAAAIAB4ioAAHJlZmluZW1lbnQxNi5weVBLBQYA
AAAABgAGAFcBAAAwMwAAAAA=
```
<!-- END COMPLEXITY16 ZIP -->

</details>
