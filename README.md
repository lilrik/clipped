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
Having to traverse through CLIP everytime I need to get some file posted by a professor is a massive pain. It's slow and requires way too much clicking around. 

We have the option of making shortcuts but, even then, it's still annoying to deal with.

Furthermore, the notification system is ancient.

## How
1. Go to `docs/user.json` and set your CLIP credentials (username and password only) in their respective fields. (**Don't change the number field**.)
2. Run your platform's executable (see releases) from inside the `clipped` directory or use the flags if from anywhere else. (See example below.)

###### (The binaries are not in the repo by default because they're too big.)

## Run
Once again, you **must** run it from inside the clipped directory or:
1. Compile it yourself and run it from anywhere with the "-embed=true" flag.
2. Run it from anywhere with the "-config=relative/path -files=relative/path" flags.

```bash
# usage: <executable> <class-name> <year>
# the <class-name> is the name in the classes.json file
# the <year> is the last two digits only (2022 -> 22)

# ex.: getting the IA 2022 class files:
./clipped-linux ia 22

# ex.: getting the IA 2022 class files whilst running from "some-dir" inside the "clipped" directory:
./clipped-linux -config="../config" -files=".." ia 22

# flag descriptions
./clipped-linux -h
```

## Build
This project has no external dependencies so running `go build .` is enough. (Don't forget to set the flags if running from outside root.)

## Add more classes
1. Go to the class' CLIP page and, from the URL, copy the number in the `unidade=XXXX` field.
2. Go to `docs/classes.json` and put the number, class name (whatever you want) and semester (`1` for winter or `2` for summer).

## Disclaimer
Please don't abuse the servers and I'm not responsible if you do.
