# Calendario per i corsi Unibo V2

Il progetto mira a fornire un calendario in formato ICS per i corsi dell'Università di Bologna, in modo da poterli
aggiungere al proprio calendario personale.

## Build

E' necessario avere installati

- go (versione 1.20 o superiore)
- pnpm o simili

Per compilare il progetto eseguire

```bash
pnpm install
pnpm run css:build
go build
```

Il file generato (`unibocalendar`) contiene tutto il necessario per l'esecuzione del programma.

## Deploy

Creare una cartella dove spostare l'eseguibile e dopo eseguirlo:

```bash
./unibocalendar
```

Per eseguire in modalità release

```bash
GIN_MODE=release ./unibocalendar
```

Il server verrà avviato su http://localhost:8080.

## Utilizzo

Per ottenere il calendario di un corso andare su http://localhost:8080/courses/ (o <url del server>/courses) e
selezionare l'anno di frequenza e il corso di interesse.

Copiare il collegamento che viene fornito e aggiungerlo al proprio calendario.





