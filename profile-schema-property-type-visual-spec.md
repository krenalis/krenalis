# Profile schema property type — specifica visuale

Questa specifica descrive il controllo `Type` nel pannello delle property, la sua rappresentazione nella grid e gli stati relativi alle property nuove o già materializzate.

Non descrive l'implementazione corrente: è il riferimento per l'implementazione e la validazione visuale.

## Modello mostrato all'utente

La label del controllo resta `Type`.

Il primo selettore determina la struttura:

- `one value`
- `array of`
- `object`
- `map of`

Il secondo selettore determina una combinazione predefinita di physical type, semantic e, quando necessaria, semantic option. Nel profile schema editor questa combinazione viene chiamata amichevolmente formato. La famiglia fisica di ciascun formato con semantic è fissa: non si seleziona un semantic per poi abbinarlo liberamente a un'altra famiglia. Per una property nuova, il prodotto applica anche la configurazione fisica completa prevista dal formato. Per una property materializzata, il physical type rimane immutabile e il semantic può soltanto essere mantenuto esattamente o rimosso.

Il menu presenta sempre il catalogo completo. Per una property materializzata, sono selezionabili soltanto il physical
type puro corrente e, se presente nello schema applicato, il semantic originale. Le altre voci restano visibili ma
non selezionabili, con un'attenuazione lieve e senza hover.

`Array` e `Map` applicano il formato rispettivamente all'elemento e al valore. `Object` non ha un semantic diretto e non mostra il secondo selettore.

Il semantic formatted date and time non è disponibile nel profile schema editor.

Il menu aperto, il controllo chiuso e la grid usano la stessa grammatica inline e occupano una sola riga:

```text
primary label · secondary metadata
```

La quantità di dettaglio cambia intenzionalmente tra i tre contesti:

- il menu aperto aiuta a scegliere e mostra il preset con il livello di dettaglio definito dal catalogo;
- il controllo chiuso aiuta a riconoscere la selezione e usa label compatte;
- la grid permette di ispezionare la configurazione effettiva completa.

Quando esiste un semantic, questo è sempre la primary label e il physical type è sempre metadata tecnico. Primary
label e metadata tecnico usano il monospace. Senza semantic, viene mostrato il physical type, sempre in monospace.
Solo il menu aperto aggiunge una descrizione in font di sistema ai type puri. Currency, unità di measurement e unità
di duration vengono aggiunte alla primary label soltanto nella grid, dove servono a ispezionare lo schema effettivo.

Nelle label compatte, una Semantic Option viene omessa soltanto quando può essere dedotta senza ambiguità dal physical
type e dalle constraint mostrate. Se due varianti dello stesso semantic produrrebbero la stessa label, il qualificatore
dell'option resta nella primary label.

Esempi:

```text
string · Text value, such as a name or code
```

```text
URL · string
```

```text
percentage · decimal(18,4)
```

`string` rimane una voce selezionabile con questo nome. Non vengono introdotti alias fittizi come `Text` e non viene
mostrata la dicitura `No semantic`.

## Controllo chiuso

Il controllo mantiene i due segmenti già presenti e la label `Type`:

```text
┌──────────────────┬──────────────────────────────────────┐
│ ① one value   ▾  │ string                           ▾ │
└──────────────────┴──────────────────────────────────────┘
```

Con semantic:

```text
┌──────────────────┬──────────────────────────────────────┐
│ ① one value   ▾  │ URL · string                    ▾ │
└──────────────────┴──────────────────────────────────────┘
```

Senza una selezione, il secondo segmento mostra `Select type...` con lo stile placeholder degli altri input.

Regole visuali:

- altezza esterna: `40px`, uguale agli altri controlli principali;
- il contenuto è sempre su una sola riga e centrato verticalmente;
- nessuna icona nel secondo segmento;
- il caret è centrato verticalmente e occupa una colonna riservata, quindi il testo non può sovrapporsi;
- primary label e metadata seguono la stessa gerarchia tipografica delle voci del menu;
- il contenuto usa ellissi su una sola linea;
- il placeholder non mostra metadata;
- se il contenuto è troncato, hover e focus mostrano il valore completo in un tooltip;
- il secondo segmento continua ad aprire il menu per una property materializzata, per mostrare il catalogo e consentire
  la sola eventuale rimozione del semantic;
- anche il primo segmento continua ad aprire il proprio menu quando la struttura è immutabile; soltanto la struttura
  corrente è selezionabile, mentre le altre voci sono leggermente attenuate e non ricevono hover.

Il controllo chiuso non documenta la configurazione completa. Mostra il nome del type puro senza descrizione:

```text
string
boolean
int
float
decimal
datetime
date
time
year
uuid
json
ip
```

Per i semantic semplici mostra soltanto la famiglia fisica:

```text
email · string
phone number · string
country · string
URL · string
duration · int
money · decimal
percentage · decimal
measurement · decimal
```

`country` mantiene la stessa label compatta per entrambi i formati, perché la variante viene scelta nel sotto-menu
`Format`:

La riduzione è intenzionale: la voce di catalogo è visibile nel menu aperto e la configurazione completa è visibile
nella grid. Per `Array` e `Map`
il wrapper è già espresso dal primo segmento, quindi il secondo mostra soltanto il formato dell'elemento o del valore:

```text
[ array of ▾ ] [ email · string ▾ ]
```

## Menu aperto

### Struttura generale

```text
┌─────────────────────────────────────────────────────────┐
│ string · Text value, such as a name or code               │
│ URL · string                                            │
│ ...                                                     │
└─────────────────────────────────────────────────────────┘
```

Il menu:

- ha una larghezza pari al `120%` dell'intero controllo `Type`, nei limiti dello spazio disponibile;
- mostra dieci voci complete e metà della successiva per rendere evidente lo scroll verticale; riduce la propria altezza
  quando lo spazio disponibile è inferiore e usa una leggerissima ombra interna sul bordo inferiore per suggerire la
  continuazione;
- non mostra un campo di ricerca;
- non usa intestazioni come `Common`, `Basic values`, `Text`, `Numbers`, `Semantic types` o `Physical types`;
- usa divider leggeri tra gruppi visivi, senza label;
- non mostra icone nelle voci;
- mostra un check allineato a destra sulla voce selezionata;
- non nasconde mai gruppi o voci soltanto perché il physical type della property è immutabile.

### Voce del menu

Ogni voce ha altezza fissa di `32px` e mostra tutte le informazioni su una sola riga.

Voce senza semantic:

```text
┌─────────────────────────────────────────────────────────┐
│ string · Text value, such as a name or code          ✓   │
└─────────────────────────────────────────────────────────┘
```

Voce con semantic:

```text
┌─────────────────────────────────────────────────────────┐
│ URL · string                                       ✓   │
└─────────────────────────────────────────────────────────┘
```

Regole:

- padding orizzontale `12px`;
- il contenuto è centrato verticalmente;
- primary label, separatore `·` e metadata sono elementi distinti; il separatore usa un colore intermedio tra i due
  testi e ha spazio aggiuntivo su entrambi i lati;
- il semantic usa il monospace, `14px`, peso normale e colore primario;
- quando la primary label è un physical type puro, usa il monospace, `14px` e il colore primario;
- il physical type usato come metadata è in monospace, `12px` e usa `var(--text-light)`;
- la descrizione usata come metadata è in font di sistema, `13px`, peso normale e usa `var(--text-light)`;
- il metadata non deve essere tanto chiaro da compromettere leggibilità e contrasto;
- la voce non va mai a capo e usa ellissi quando necessario; non mostra tooltip al passaggio del mouse;
- hover e focus usano lo stesso background neutro delle altre voci del prodotto;
- la voce selezionata usa il background selezionato già presente nel profile schema editor;
- il divider è `1px`, usa il colore dei bordi esistenti e separa i gruppi visivi senza creare una card per ciascuna
  voce;
- divider e spaziatura tra gruppi restano esterni all'area cliccabile di `32px`.

### Catalogo completo per una property nuova o modificabile

Questo è l'ordine completo. Le righe separate da uno spazio appartengono allo stesso gruppo visivo; la linea indica
un divider. I gruppi aiutano a scorrere il catalogo e non rappresentano necessariamente una singola famiglia fisica.

```text
email · string
phone number · string
country · string
URL · string
string · Text value, such as a name or code
─────────────────────────────────────────────────────────
boolean · True or false
─────────────────────────────────────────────────────────
duration · int
int · Number with no decimal places
─────────────────────────────────────────────────────────
float · Number with approximate precision
─────────────────────────────────────────────────────────
money · decimal(18,4)
percentage · decimal(18,4)
measurement · decimal(18,4)
decimal · Decimal number with fixed precision
─────────────────────────────────────────────────────────
datetime · Date and time
date · Date without a time
time · Time without a date
year · Year
─────────────────────────────────────────────────────────
uuid · UUID
json · JSON value
ip · IPv4 or IPv6 address
```

Per un formato con semantic, il metadata mostra la configurazione fisica necessaria a comprendere la scelta. Il
catalogo omette dettagli determinati dalle Semantic Options: l'unica voce `country` mostra `string`, mentre `duration`
mostra `int` senza il bit size. La selezione applica comunque la configurazione completa indicata nella tabella
seguente. I relativi controlli del pannello compaiono soltanto quando viene selezionato un type puro, con l'eccezione
di Min e Max per `money`, `percentage` e `measurement`.

Per una property non materializzata, il menu aperto conserva sempre queste label canoniche, anche dopo che l'utente
ha modificato le constraint di un type puro. Il controllo chiuso mostra invece soltanto la label compatta del formato,
per esempio `decimal` per un type puro o `percentage · decimal` per il semantic `percentage`.

`Object`, `Array` e `Map` non compaiono nel secondo menu perché sono scelte del primo selettore.

### Configurazione fisica per una nuova property

La selezione di un formato crea il relativo physical type con la configurazione stabilita dal prodotto:

| Formato                         | Physical type fisso | Type creato alla selezione                                                           |
| ------------------------------- | ------------------- | ------------------------------------------------------------------------------------ |
| `email`, `phone number` e `URL` | `string`            | `string` senza constraint                                                            |
| `country · string`              | `string`            | `string, max 2 chars` per default; il formato a 3 lettere crea `string, max 3 chars` |
| `money · decimal(18,4)`         | `decimal`           | `decimal(18,4)` senza constraint Min o Max                                           |
| `percentage · decimal(18,4)`    | `decimal`           | `decimal(18,4)` senza constraint Min o Max                                           |
| `measurement · decimal(18,4)`   | `decimal`           | `decimal(18,4)` senza constraint Min o Max                                           |
| `duration · int`                | `int`               | `int(64)`                                                                            |

La selezione dei type puri conserva i valori iniziali già usati dall'editor; questa specifica non li modifica. Dopo
la selezione, i controlli del physical type restano visibili e configurabili.

Quando è selezionato un semantic, i controlli specifici del physical type non vengono mostrati e la configurazione
gestita dal prodotto non è modificabile. Per `money`, `percentage` e `measurement`, Precision e Scale restano nascoste
e fissate a 18 e 4, mentre Min e Max sono opzionali e modificabili finché la property non è materializzata. Selezionare
un type puro rimuove il semantic, mantiene la configurazione fisica corrente e, se la property non è materializzata,
rende disponibili tutti i relativi controlli.

La compatibilità più ampia accettata dal package `tools/types` non viene esposta dal profile schema editor.
`tools/types` ammette `percentage` su qualsiasi `decimal` valido e ammette `money` e `measurement` anche sugli altri
type numerici compatibili; non impone precisione, scala o range. Soltanto i profile schema richiedono `decimal(18,4)`
per tutti e tre i semantic.

### Property già materializzata

Il menu conserva il catalogo completo. Cambia soltanto la disponibilità delle voci.

La helper tra la label e il controllo è mostrata non appena una property ha un Type selezionato. Il nome del base type
è monospace. Prima della materializzazione, per una property senza semantic:

```text
Can't be changed once the property has been applied.
```

Prima della materializzazione, per una property con semantic:

```text
Once applied, this type can only be changed to <base type>.
```

Dopo la materializzazione, per una property senza semantic:

```text
This type can't be changed.
```

Dopo la materializzazione, per una property con semantic:

```text
This type can only be changed to <base type>.
```

Regole:

- la voce del physical type corrente senza semantic è sempre abilitata;
- se la property materializzata ha un semantic, rimane abilitata anche e soltanto la voce di quel semantic originale;
- il semantic originale comprende tutte le sue options: currency, format e unità non possono essere aggiunti o cambiati;
- se il semantic viene rimosso nella bozza, la sua voce rimane abilitata per consentire di annullare la rimozione prima
  di `Apply changes`; riselezionarla ripristina anche le options originali;
- una property materializzata senza semantic non può acquisirne uno;
- non è possibile sostituire un semantic con un altro;
- gli altri physical type e formati restano visibili ma non sono selezionabili;
- una voce non selezionabile è soltanto leggermente attenuata, non riceve hover e non mostra il check; conserva il
  focus da tastiera previsto dal pattern ARIA del menu, che la annuncia come non disponibile senza renderla attivabile;
- tutte le voci mantengono l'ordine canonico del catalogo;
- ciascun formato appare una sola volta;
- per una voce semantic abilitata, il metadata mostra il physical type effettivo della property con le sue constraint,
  non la configurazione usata per creare una property nuova;
- per la voce del type puro abilitata, la primary label mostra il physical type effettivo con le sue constraint;
- le voci non selezionabili mantengono le label canoniche del catalogo;
- la ripetizione del physical type effettivo nelle voci abilitate è intenzionale: rende esplicito che rimuovere semantic
  non modifica la configurazione materializzata.

Il metadata tecnico di una property materializzata mostra esclusivamente la configurazione fisica realmente presente.
Non aggiunge mai precisione, bit size o constraint provenienti dal preset usato per creare una property nuova con quel
semantic.

Rimuovere il semantic non converte il physical type. Elimina soltanto il semantic e le sue options, preservando
esattamente struttura, bit size, segno, precisione, scala, lunghezza, Min e Max. La modifica riguarda esclusivamente i
metadata e non genera operazioni DDL sul data warehouse.

Esempi:

```text
email · string
→ string

country — 2-letter ISO code · string · max 2 chars
→ string · max 2 chars

money · decimal(18,4) · min -100, max 100
→ decimal(18,4) · min -100, max 100

duration — s · int(64)
→ int(64)
```

Esempio per una property esistente `string, max 100 chars`:

```text
This type can't be changed.

email · string                                             not selectable
phone number · string                                      not selectable
country · string                                           not selectable
URL · string                                               not selectable
string · max 100 chars · Text value, such as a name or code
─────────────────────────────────────────────────────────
boolean · True or false                                    not selectable
─────────────────────────────────────────────────────────
duration · int                                             not selectable
int · Number with no decimal places                        not selectable
─────────────────────────────────────────────────────────
float · Number with approximate precision                 not selectable
─────────────────────────────────────────────────────────
money · decimal(18,4)                                     not selectable
percentage · decimal(18,4)                                not selectable
measurement · decimal(18,4)                               not selectable
decimal · Decimal number with fixed precision             not selectable

...
```

Se la stessa property avesse il semantic `email`, soltanto `string` ed `email` sarebbero abilitate:

```text
This type can only be changed to string.

email · string · max 100 chars
phone number · string                                      not selectable
country · string                                           not selectable
URL · string                                               not selectable
string · max 100 chars · Text value, such as a name or code
...
```

Esempio per una property esistente `int(64)`:

```text
This type can't be changed.

email · string                                             not selectable
phone number · string                                      not selectable
country · string                                           not selectable
URL · string                                               not selectable
string · Text value, such as a name or code                not selectable
...
─────────────────────────────────────────────────────────
boolean · True or false                                    not selectable
─────────────────────────────────────────────────────────
duration · int                                             not selectable
int(64) · Number with no decimal places
─────────────────────────────────────────────────────────
float · Number with approximate precision                 not selectable
─────────────────────────────────────────────────────────
money · decimal(18,4)                                     not selectable
percentage · decimal(18,4)                                not selectable
measurement · decimal(18,4)                               not selectable
decimal · Decimal number with fixed precision             not selectable
...
```

Per `Array` e `Map`, la stessa regola si applica al semantic dell'elemento o del valore, mentre il wrapper e il relativo
physical type completo restano invariati.

## Controlli successivi nel pannello

I controlli compaiono immediatamente sotto `Type`, prima di `Display name (optional)` e `Description (optional)`.

### Constraint del physical type

Non viene aggiunto alcun controllo `Storage type`. Il physical type è parte del formato scelto nel menu.

Quando viene selezionato un type puro, vengono mostrati i relativi controlli del physical type, come lunghezza della
stringa, bit size e segno degli integer, precisione e scala dei decimal, Min e Max. Questi controlli sono configurabili
soltanto finché la property non è materializzata.

Quando viene selezionato un semantic su una property non ancora materializzata, il prodotto applica la configurazione
fisica prevista dal formato e nasconde i controlli che potrebbero modificarla. Restano visibili soltanto gli eventuali
controlli specifici del semantic elencati nella sezione seguente e, per `money`, `percentage` e `measurement`, Min e
Max opzionali.

Per una property materializzata, il semantic può soltanto essere mantenuto o rimosso. La helper tra la label e il controllo
comunica che la configurazione fisica è fissa; gli eventuali controlli del physical type sono mostrati soltanto quando
è selezionato il type puro e sono read-only. Conservano il normale contrasto perché i valori descrivono la
configurazione materializzata, non entrano nel tab order e usano il cursore `not-allowed` per comunicare che non
possono essere modificati.

### Opzioni dei semantic

| Formato selezionato per una nuova property | Controllo successivo                                                        |
| ------------------------------------------ | --------------------------------------------------------------------------- |
| `email · string`                           | nessuno                                                                     |
| `phone number · string`                    | nessuno                                                                     |
| `country · string`                         | select `Format`: `2-letter ISO code` per default oppure `3-letter ISO code` |
| `URL · string`                             | nessuno                                                                     |
| `money · decimal(18,4)`                    | select `Currency`, poi input opzionali `Min` e `Max`                        |
| `percentage · decimal(18,4)`               | badge con testo `0.9 represents 90%` e input opzionali `Min` e `Max`        |
| `measurement · decimal(18,4)`              | select obbligatorio `Unit`, poi input opzionali `Min` e `Max`               |
| `duration · int`                           | select obbligatorio `Unit`                                                  |

Per `money`, `percentage` e `measurement`, Precision e Scale non sono modificabili e non vengono mostrate. Min e Max
non hanno valori predefiniti e possono essere lasciati vuoti; devono soltanto rispettare i limiti propri di
`decimal(18,4)`. Per una property materializzata vengono mostrati soltanto quando è selezionato uno dei tre semantic e
sono read-only.
Per una property materializzata sono read-only anche `Format`, `Currency` e `Unit`: le Semantic Options applicate non
possono essere riconfigurate, ma i rispettivi valori rimangono visibili con il normale contrasto.
Per `percentage`, il testo `0.9 represents 90%` compare in un badge neutro con pill shape, coerente con gli altri badge
del profile schema editor, immediatamente sotto il secondo segmento del controllo `Type` e separato dal menu di `8px`.
Il badge usa lo sfondo `#ededf7`, occupa la larghezza del secondo segmento lasciando lo stesso margine a sinistra e a
destra e mostra il testo centrato con peso normale. Il testo usa il precedente grigio
`var(--sl-color-neutral-700)` con una tinta del `12%` verso `#6062d0`. Il badge è alto `19px` e precede i controlli
Min e Max. `money` e `measurement` non mostrano alcun badge.

`Currency` precede i controlli Min e Max. Quando non è impostata, il select mostra `No currency specified`; la stessa
voce permette di rimuovere una selezione esistente. Le valute più comuni compaiono per prime in questo ordine:
`USD · US dollar`, `EUR · Euro`, `GBP · British pound`, `JPY · Japanese yen`, `CNY · Chinese Yuan`. Quando esiste un
simbolo distinto dal codice, viene mostrato a destra della voce. Le altre valute seguono in ordine alfabetico.

Il menu `Unit` di measurement usa heading piccoli, bold e non selezionabili. Un divider leggero separa ciascun gruppo
dal successivo:

```text
Length
Millimetre · mm
Centimetre · cm
Metre · m
Kilometre · km
Inch · in
Foot · ft
Yard · yd
Mile · mi

Weight
Gram · g
Kilogram · kg
Ounce · oz
Pound · lb

Data size
Byte · B
Kilobyte · kB
Megabyte · MB
Gigabyte · GB

Temperature
Degree Celsius · °C
Degree Fahrenheit · °F

Volume
Millilitre · mL
Litre · L
```

`Unit` per duration non seleziona silenziosamente un valore iniziale: mostra il placeholder `Select a unit` e richiede una
scelta prima della conferma. L'eventuale errore viene mostrato sotto questo controllo.

```text
Milliseconds · ms
Seconds · s
Minutes · min
Hours · h
Days · d
Weeks · wk
```

## Grid del profile schema

La colonna resta `Type`; non viene aggiunta una colonna `Semantic`. La colonna `Description` viene rimossa per lasciare
più spazio al type completo. Il display name continua a essere mostrato sotto il nome della property, mentre la
description resta consultabile e modificabile nel pannello laterale.

La grid mostra tutte le informazioni su una sola riga. Quando esiste un semantic, questo è la primary label e il
physical type completo della property è il metadata tecnico:

```text
email · string
```

```text
URL · string
```

```text
country — 2-letter ISO code · string · max 2 chars
country — 3-letter ISO code · string · max 3 chars
```

```text
duration — s · int(64)
duration — min · int(64)
```

```text
money · decimal(18,4)
money — EUR · decimal(18,4) · min -100.25, max 999.9999
```

```text
percentage · decimal(18,4)
percentage · decimal(18,4) · min -0.5, max 1.5
```

```text
measurement — kg · decimal(18,4)
measurement — kg · decimal(18,4) · min -10.5, max 10.5
```

Per un type puro viene mostrata soltanto la configurazione effettiva, senza le descrizioni usate nel menu aperto:

```text
string · max 100 chars

unsigned int(32) · min 0

decimal(12,4)

boolean
```

Regole visuali:

- altezza di ogni riga della grid fissa a `54px`;
- il contenuto della cella `Type` è centrato verticalmente;
- il semantic usa il monospace e il colore primario;
- il physical type dopo il separatore `·` usa il monospace e un colore secondario;
- un type puro usa il monospace e il colore primario;
- semantic, separatore e physical type sono span distinti;
- il contenuto resta su una sola linea, con ellissi e tooltip se troncato;
- la freccia di espansione degli object resta centrata rispetto ai `54px` della riga e spostata di `1px` verso il basso, come già deciso per la grid;
- lo stesso rendering viene usato nella grid di consultazione e in quella di modifica.

La variante `country` viene mostrata soltanto dove serve ispezionare la configurazione effettiva:

- menu aperto: un'unica voce `country · string`;
- controllo chiuso: `country · string` per entrambi i formati;
- grid: `country — 2-letter ISO code · string · max 2 chars` e
  `country — 3-letter ISO code · string · max 3 chars`.

Per una property materializzata, la grid conserva lo stesso qualificatore e mostra il physical type effettivo senza
aggiungere i constraint dei preset:

```text
country — 2-letter ISO code · string · max 100 chars
country — 3-letter ISO code · string · max 100 chars
```

La stessa riduzione contestuale viene applicata a `money`, `percentage` e `measurement`: il menu aperto mostra il
preset `decimal(18,4)`, la grid mostra anche gli eventuali Min e Max effettivi e il controllo chiuso mostra soltanto
la rispettiva label semantic seguita da `· decimal`.

La currency di `money`, quando presente, compare nella primary label della grid come codice canonico. Le unità di
measurement e duration compaiono entrambe con il simbolo canonico.

Nel secondo picker il type mostrato è sempre quello dell'elemento di un array o del valore di una map, perché la
struttura è già esplicitata dal primo controllo. Nella grid la struttura precede invece il formato completo:

```text
array of email · string
```

Per `Map` viene usata la forma analoga:

```text
map of money · decimal(18,4)
```

I due contesti descrivono livelli diversi della struttura, quindi questa differenza non è un'incoerenza.

## Pannello di dettaglio in sola lettura

Il campo `Type` del pannello in sola lettura mantiene una composizione compatta a due righe distinta dalla grid. La
prima riga omette bit size ed eventuali constraint, come precisione, scala, lunghezza e range. Mantiene invece il
segno degli integer e la struttura completa della property, quindi può mostrare `unsigned int`, `array of string` o
`map of decimal`.

Questo differisce dal secondo selettore del menu `Type`, che mostra soltanto il physical type dell'elemento di un
array o del valore di una map, perché la struttura `array of` o `map of` è già mostrata dal primo selettore.

```text
decimal
percentage
```

Il pannello non introduce un campo `Semantic` separato. `Currency` e `Unit` vengono mostrate in righe dedicate quando
presenti, con label coerenti con i controlli di modifica:

- `Currency`;
- `Unit`.

Il formato di `country` resta invece riconoscibile nella seconda riga compatta del campo `Type`, per esempio
`country (2-letter)`.

## Stati da validare visivamente prima di considerare definitiva l'implementazione

La UI deve essere verificata almeno nei seguenti stati, prima di considerare definitiva l'implementazione:

1. property nuova senza type, con il menu completo aperto, verificando ordine, divider, voci su una sola riga,
   gerarchia tra primary label e metadata, altezza di `32px`, assenza di icone e assenza del semantic formatted date
   and time;
2. property nuova con `string` puro, verificando la label compatta `string` nel controllo chiuso e la presenza dei
   controlli del physical type;
3. property nuova con `URL` e con `country`, verificando il default a due lettere del sotto-menu `Format`, il cambio
   al formato a tre lettere, i rispettivi preset `string, max 2 chars` e `string, max 3 chars` nella grid, la label
   compatta `country · string` nel controllo chiuso e l'assenza dei controlli del physical type;
4. property nuove con `money`, `percentage` e `measurement`, verificando il preset `decimal(18,4)`, le label compatte,
   l'assenza dei controlli Precision e Scale, la presenza di Min e Max opzionali, il badge soltanto su `percentage`,
   il select `Currency` prima del range e i controlli specifici delle unità;
5. property nuova con `duration`, verificando `int(64)`, il placeholder obbligatorio di `Unit` e l'assenza dei
   controlli del physical type gestiti dal prodotto;
6. property esistente `string, max 100 chars` senza semantic, con helper sul Type, ordine canonico del catalogo,
   soltanto `string` selezionabile e tutti i semantic non selezionabili ma solo leggermente attenuati;
7. property esistente `email · string, max 100 chars`, con soltanto `string` ed `email` abilitati; dopo aver rimosso il
   semantic, verificare i controlli della stringa read-only, le constraint di lunghezza invariate e la possibilità
   di ripristinare `email` fino a `Apply changes`;
8. property esistente `decimal(12,4)`, con `money`, `percentage` e `measurement` visibili nelle rispettive posizioni
   canoniche ma non selezionabili;
9. property esistente `money · EUR · decimal(18,4), min -0.5, max 1.5`, con soltanto `decimal` e `money` abilitati,
   Currency, Min e Max read-only e ripristino della currency originale quando si annulla la rimozione del semantic;
10. property esistente `float`, con il solo physical type puro abilitato e nessun formato semantic disponibile;
11. `array of string` con `email`, verificando `email · string` nel controllo chiuso e `array of email · string` nella
    grid;
12. `map of decimal(18,4)` con `money`, verificando `money · decimal` nel controllo chiuso e
    `map of money · decimal(18,4)` nella grid;
13. `object`, senza secondo selettore;
14. pannello di modifica stretto da `340px`, verificando caret, ellissi e tooltip;
15. grid con righe miste con e senza semantic, verificando il layout inline, la configurazione fisica completa,
    l'assenza delle descrizioni dei type puri, altezza e centratura;
16. pannello di dettaglio in sola lettura, verificando physical type compatto e le righe dedicate a currency, unità
    di misura e unità di durata.

Questi stati definiscono la copertura visuale da verificare, ma non richiedono un test automatico distinto per ciascun
caso. I test devono concentrarsi sui comportamenti che possono regredire senza essere rilevati dal type checking o da
una revisione visuale ordinaria.

## Criteri non negoziabili per il prossimo prototipo

- Il menu aperto mostra tutti i physical type anche per una property esistente.
- `string` è selezionabile direttamente e non viene rinominato `Text`.
- Non ci sono category heading.
- Non ci sono icone nel secondo selettore.
- Non c'è un campo di ricerca nel menu.
- Tutte le voci hanno altezza fissa di `32px`.
- Il menu aperto, il controllo chiuso e la grid mostrano il contenuto della colonna `Type` su una sola riga.
- Il semantic non causa una nuova colonna nella grid.
- Il menu aperto mostra il catalogo completo; il controllo chiuso usa label compatte; la grid mostra la configurazione
  effettiva completa.
- Il pannello in sola lettura conserva la propria composizione compatta a due righe.
- Il physical type non viene nascosto quando è presente un semantic.
- Per una property materializzata, il metadata non aggiunge configurazioni o constraint provenienti dal preset del
  semantic.
- Per una property materializzata, semantic e relative options possono soltanto restare identici o essere rimossi; non
  possono essere aggiunti, sostituiti o riconfigurati.
- La rimozione del semantic conserva esattamente il physical type e non genera DDL sul data warehouse.
- Una Semantic Option viene omessa dalle label compatte soltanto quando è deducibile senza ambiguità dal metadata
  fisico mostrato.
- Ogni formato con semantic determina una famiglia fisica e, per una property nuova, una configurazione completa; non
  esiste un selettore separato per riabbinarlo a un'altra famiglia.
- I controlli del physical type vengono mostrati soltanto per i type puri; con un semantic, la configurazione fisica
  è gestita dal prodotto. `money`, `percentage` e `measurement` espongono soltanto Min e Max opzionali, senza rendere
  modificabili Precision e Scale.
- L'ordine del catalogo non cambia per una property materializzata.
- Nessuna logica viene considerata approvata soltanto perché compare nel checkpoint o in un prototipo precedente.
