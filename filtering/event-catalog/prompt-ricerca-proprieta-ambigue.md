# Ricerca mirata sulle proprietà ambigue del catalogo eventi Krenalis

Stiamo traducendo un catalogo di eventi derivato dalle raccomandazioni di
Segment e RudderStack negli schemi nativi di Krenalis. Ti chiediamo una ricerca
sulle fonti ufficiali per risolvere le ambiguità elencate sotto, prima di
scegliere alcuni tipi e semantiche delle proprietà.

Non serve riprogettare il catalogo, il sistema di tipi o il motore di filtering.
Non assumere di avere accesso al repository. Questo prompt contiene il contesto
e i riferimenti necessari; se hai anche i tre artefatti originali, consultali.

## Contesto e criteri già concordati

- Il catalogo contiene 102 definizioni. Il riconoscimento usa i matcher originali:
  per i casi track sotto, `type = track` e `event = nome esatto`.
- `cdp-event-catalog.json` è autorevole per definizioni e matcher.
- `cdp-event-properties.json` è autorevole per proprietà, descrizioni, tipi ed
  evidenze delle fonti. I JSON Schema dei payload sono derivati automaticamente.
- La base è il profilo `documented`: un tipo soltanto suggerito o ricavato da un
  esempio non diventa automaticamente un requisito universale.
- Il tipo Krenalis viene scelto secondo il significato del dato: String, Boolean,
  Int, Float, Decimal, DateTime, Date, Time, Year, UUID, IP, JSON, Object, Array o Map.
  Sono disponibili semantiche come Money, Duration, Measurement e Percentage.
- Gli importi monetari possono usare Decimal(18,4) con semantica Money. La valuta
  non viene fissata nel tipo quando può variare ed è indicata da un campo separato.
- Un valore numerico accettato anche come stringa può avere un tipo numerico
  Krenalis. Un campo testuale accettato anche come numero può avere tipo String.
  Una struttura realmente eterogenea può richiedere JSON. Queste scelte devono
  dipendere dal significato documentato, non dalla sola presenza di un'unione.
- Le conversioni del decoder sono ancora da definire. Le regole dei mapping sono
  un riferimento da valutare, non una decisione vincolante per gli eventi.
- I tipi scelti devono essere utili anche alla futura rappresentazione in colonne
  warehouse. Non cerchiamo di accettare ogni valore teoricamente ammesso da JSON
  Number. Non è necessario introdurre un nuovo tipo Number.
- Gli array di oggetti sono ammessi negli schemi degli eventi. Non occorre
  appiattirli o convertirli in testo.

Per ogni conclusione distingui ciò che la fonte prescrive, ciò che mostra soltanto
in un esempio e ciò che sarebbe una nostra scelta locale. Se le fonti non
risolvono una domanda, dichiaralo: non inventare un contratto comune ai provider.

## 1. Product Reviewed: significato di rating

Evento: `commerce.product_reviewed`, nome tecnico `Product Reviewed`.
Proprietà: `properties.rating`.

Il catalogo riporta String nelle tabelle ecommerce dei due provider, ma un esempio
ufficiale Track di RudderStack contiene il numero `3.0`. L'unione risultante è
String/Number; non sono documentati una scala, un intervallo o un vocabolario chiuso.

Fonti riportate nel catalogo:

- https://www.rudderstack.com/docs/event-spec/ecommerce-events-spec/reviewing/#product-reviewed
- https://www.twilio.com/docs/segment/connections/spec/ecommerce/v2#product-reviewed
- https://www.rudderstack.com/docs/event-spec/standard-events/track/#sample-payload

Verifica se rating sia necessariamente numerico, con una rappresentazione
testuale alternativa, oppure se siano ammesse valutazioni testuali non numeriche.
La scala è realmente libera? Le fonti giustificano Int, Decimal o String, oppure
la scelta deve restare una convenzione Krenalis? Non dedurre un range 1–5 dal
solo nome del campo o dal valore 3.0 dell'esempio.

## 2. Video Playback Started: ad_type singolo oppure collezione

Evento: `video.playback_started`, nome tecnico `Video Playback Started`.
Proprietà: `properties.ad_type`.

Le tabelle dichiarano String; esempi ufficiali dei due provider riportano Array.
Il catalogo conserva entrambe le rappresentazioni. La descrizione attuale parla
di categoria di collocazione della pubblicità associata al record di playback.

Fonti riportate nel catalogo:

- https://www.twilio.com/docs/segment/connections/spec/video
- https://www.rudderstack.com/docs/event-spec/standard-events/video-events-spec/

Verifica il significato specifico del campo in Playback Started: una categoria,
una lista delle categorie previste, oppure una lista allineata ad altri campi
come ad_asset_id e ad_pod_id? Gli elementi dell'array hanno un tipo esplicitamente
documentato? La differenza rispetto agli altri eventi di playback è intenzionale
o emerge soltanto da esempi incoerenti?

Non raccomandare di serializzare automaticamente un array in String o di
trasformare ogni String in un array: sarebbero conversioni aggiuntive da decidere.

## 3. Video: durata e posizione temporale, interi o frazionari

Evento principale: `video.playback_seek_started`, nome tecnico
`Video Playback Seek Started`; proprietà `properties.seek_position`.

Il catalogo indica un offset in secondi. RudderStack dichiara Integer; Segment
non esplicita il tipo nel testo considerato, ma mostra un Number in un esempio.
Il testo RudderStack richiama sia Seek Started sia Seek Completed in modo non
uniforme. Non è stata introdotta una relazione obbligatoria fra i due eventi.

Usa le fonti video del punto precedente per verificare:

- se gli offset e le durate pertinenti accettino frazioni di secondo;
- quali campi siano durate e quali posizioni lungo la timeline;
- se l'unità sia esplicita per ciascun campo e coincida fra i provider;
- l'applicabilità e il significato di seek_position nei due eventi di seek.

Serve sapere se una rappresentazione intera in una unità più fine possa
preservare i valori ammessi, oppure se le fonti lascino aperta la precisione.
Non scegliere una precisione universale soltanto dagli esempi.

È già noto che `total_length` ammette null per una sessione live: non confonderlo
con zero, assenza del campo o una durata sconosciuta di un video finito. La
gestione Krenalis della nullability è una decisione separata; questo prompt non
autorizza a eliminare quel significato per rispettare i limiti dei profili.

## 4. Product Added e Product Removed: quantità e significato della variazione

Proprietà: `properties.quantity`. Il tipo documentato è Number. Per Product
Added il catalogo non stabilisce se si tratti della quantità aggiunta oppure
della quantità risultante della riga nel carrello.

Fonti iniziali per Product Added:

- https://www.rudderstack.com/docs/event-spec/ecommerce-events-spec/ordering/#product-added
- https://www.twilio.com/docs/segment/connections/spec/ecommerce/v2#product-added

Verifica anche le sezioni ufficiali Product Removed. Le fonti consentono quantità
frazionarie? Distinguono quantità della variazione e quantità risultante? In caso
di rimozione il numero è positivo con segno implicito nell'evento, oppure può
essere negativo? Distingui le dichiarazioni dagli esempi di articoli venduti
in unità intere. Non estendere automaticamente le risposte agli altri eventi.

## 5. Live Chat: conversation_duration all'inizio della conversazione

Eventi: `chat.conversation_started`, `chat.conversation_ended`,
`chat.message_sent`, `chat.message_received`.
Proprietà: `properties.conversation_duration`.

Il catalogo associa a Segment un Number in secondi. Il campo compare anche su
Conversation Started e sui messaggi; per RudderStack non viene asserito un
contratto corrispondente nelle fonti esaminate.

Fonte iniziale:

- https://www.twilio.com/docs/segment/connections/spec/live-chat

Verifica se il campo sia effettivamente previsto per tutti questi eventi e che
cosa misuri in ciascuno, in particolare all'inizio. Sono ammessi secondi
frazionari? Se la pagina ripete una tabella generica senza chiarire questi aspetti,
segnala il limite anziché costruire un significato più preciso.

## Risposta richiesta

Restituisci una tabella per evento e proprietà con:

1. significato accertato e differenze fra provider;
2. tipi e rappresentazioni esplicitamente documentati;
3. unità, precisione, nullability o vincoli realmente dichiarati;
4. esempi e contraddizioni che non costituiscono da soli un contratto;
5. tipo e semantica Krenalis consigliati, motivati e distinti dalla fonte;
6. eventuali conversioni da discutere separatamente;
7. URL ufficiali, sezione e data della verifica;
8. questioni che richiedono ancora una decisione di prodotto.

Usa fonti primarie e cita brevemente i passaggi decisivi. L'obiettivo è chiarire
i pochi campi ambigui, senza ampliare il linguaggio dei filtri né modificare le
definizioni autorevoli degli archivi.
