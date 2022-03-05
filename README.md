## What
It downloads the files (of a given year) from the class' page on CLIP and puts them in their own directory and respective subdirectories.
```bash
Outros              [1/8] ████████████████████ 100.00% (4 new files)
Material-multimédia [2/8] ████████████████████ 100.00% (22 new files)
Problemas           [3/8] (no files)
Protocolos          [4/8] (no files)
Seminários          [5/8] (no files)
Exames              [6/8] (no files)
Testes              [7/8] ████████████████████ 100.00% (3 new files)
Textos-de-apoio     [8/8] (no files)
```
```
─ ia
  ├── material-multimedia
  │   ├── T1.pdf
  │   ├── T2.pdf
  │   ├── T3.pdf
  │   └── (...)
  ├── outros
  │   ├── Avaliacao Sumativa.htm
  │   ├── GruposIA.pdf
  │   ├── pauta.html
  │   └── teste2_salas.html
  └── testes
      ├── teste1Asol.pdf
      └── teste2A_sol.pdf
```
## Why
Having to traverse through CLIP everytime I need to get some file posted by a professor is a massive pain: slow and requires way too much clicking around. We have the option of making and using shortcuts but even then it's still annoying to deal with.
Furthermore, the notification system is ancient.

## How
1. Go to `docs/user.json` and set your CLIP credentials (the ones you use to log-in) in their respective fields. **Don't change the number field**.
2. Run your platform's executable in `bin` (see the example below).

## Running
```bash
# usage: <executable> <class-name> <year>
# the <class-name> is the name in the classes.json file
# the <year> is the last two digits only
# by default the folder containing the files will go to the project root (use flags to change)

# ex.: getting the IA 2022 class files:
./clipped-linux ia 22
```

#### Flags
Run `<executable> -h` to get a short description of the available flags.

## Adding more classes
1. Go to class' CLIP page and, from the URL, copy the number in the `unidade=XXXX` field.
2. Go to `docs/classes.json` and put the number, the class name (whatever you want) and semester (1 or 2 for winter or summer, respectively) in their respective fields.

## Building
This project has no external dependencies. Just use `go build` (and don't forget to set the flags if running from root).

## Disclaimer
Please don't abuse the servers and I'm not responsible if you do.
