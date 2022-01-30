## What
It downloads the files (of a given year) from the class' page on CLIP and puts them in their own directory and respective subdirectories.
```
─ ia
  ├── material-multimedia
  │   ├── T1.pdf
  │   ├── T2.pdf
  │   ├── T3.pdf
  │   └── (a lot more...)
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
1. Go to CLIP and, from the URL, copy the number in the `aluno=XXXXXX` field.
2. Go to `docs/user.json` and put the number and your CLIP credentials (the ones you use to log-in) in their respective fields.
3. Run your platform's executable in `bin`. (See the example below).

## Running
```bash
# usage: <executable> <class-name> <year>
# the <class-name> is the name in the classes.json file
# the <year> is the last two digits only
# by default the folder containing the files will go to the project root
# ex.: here we get the IA 2022 class files
./clipped-linux ia 22
```

#### Flags
###### Note: these flags are required if you run the executable outside of `bin`.

Run `<executable> -h` to get a short description of the flags.
```bash
# ex.: here we search for the configs in the docs folder and put the downloaded files in the parent (..) directory 
./clipped-linux ia 22 -docs=docs -files=..
```

## Adding more classes
1. Go to class' CLIP page and, from the URL, copy the number in the `unidade=XXXX` field.
2. Go to `docs/classes.json` and put the number, the class name (whatever you want) and semester (1 or 2 for summer or winter, respectively) in their respective fields.

## Building
This project has no external dependencies. Just use `go build` (and don't forget to set the flags if running from root).

## Disclaimer
Please don't abuse the servers and I'm not responsible if you do.

## QA
#### Why no concurrency?
I tested with various concurrent set-ups but it appears there is quite a strict rate-limit and I'm really not confident in CLIP's ability to not crash, even then. Don't wanna get my ass busted lmao.

#### Why do I have to fill everything myself?
It's just a one time setup but I agree; in the future I should query the database or something.
