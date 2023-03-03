## What
Automatically retrieves and organises class documents for a given year.
```bash
$ ./clipped ia 22
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
$ tree ia
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
Having to traverse through CLIP everytime I need to get some file posted by a professor is a massive pain. It's slow and requires way too much clicking around. We do have the option of making shortcuts but even then, it's still annoying to deal with.
Furthermore, the notification system is ancient.

## Run
1. Go to `docs/user.json` and set your CLIP credentials (username and password) in their respective fields. **Don't change the number field**.
2. Run your platform's executable (see releases or [compile it yourself](#build)) from inside the repo's folder ([or not](#run-from-anywhere)).

###### (The following behavior can be altered by setting certain flags.)
**By default:**
- The folder containing the files will be created in the directory you're running from.
- The folder containing the user config will be searched for as a folder named `config` in the directory you're running from.

```bash
# usage: <executable> <class-name> <year>
# the <class-name> is the name defined in "classes.json"
# the <year> is the last two digits only (2022 -> 22)

# example: getting the IA 2022 class files.
$ ./clipped ia 22
```

### Run from anywhere
###### `./clipped -h` gives a short description of the available flags.
There are two options to run the executable from anywhere:
1. Compile it yourself (with the right config) and set `-embed=true` when running. (Recommended.)
2. Manually set the `config` and `files` flag as the path to those folders **relative** from the executable.

Optionally:
```bash
# in your terminal config file (~/.bashrc for example)
export PATH=your/path/to/executable/folder:$PATH # add to path to run from anywhere
alias clipped="clipped -embed=true"              # recommended for less typing
```

## Build
This project has no external dependencies. Install Go and run `make`.

## Add more classes
1. Go to class' CLIP page and, from the URL, copy the number in the `unidade=XXXX` field.
2. Go to `docs/classes.json` and set the number, class name (whatever you want) and semester (`1` for winter, `2` for summer).

## Disclaimer
Please don't abuse the servers and I'm not responsible if you do.
