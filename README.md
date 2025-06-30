# AlmaCalendar – Calendario per i corsi Unibo V2

AlmaCalendar mira a fornire un calendario in formato ICS per i corsi dell'Università di Bologna, in modo da poterli
aggiungere al proprio calendario personale.

## Build

E' necessario avere installati

- go (versione 1.23 o superiore)
- pnpm

Per compilare il progetto eseguire

```bash
pnpm install
go generate ./...
go build
```

o più semplicemente

```bash
just
```

Il file generato (`almacalendar`) contiene tutto il necessario per l'esecuzione del programma.

## Deploy

Creare una cartella dove spostare l'eseguibile e dopo eseguirlo:

```bash
./almacalendar
```

Per eseguire in modalità release

```bash
GIN_MODE=release ./almacalendar
```

Il server verrà avviato su http://localhost:8080.

## Utilizzo

Per ottenere il calendario di un corso andare su http://localhost:8080/courses/ (o <url del server>/courses) e
selezionare l'anno di frequenza e il corso di interesse tramite AlmaCalendar.

Copiare il collegamento che viene fornito e aggiungerlo al proprio calendario.
