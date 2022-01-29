## For whom
Only useful for FCT-UNL students.

## Why
Having to traverse through CLIP everytime I need to get some file posted by a professor is a massive pain: slow and requires way too much clicking around. We have the option of making and using shortcuts but even then it's still annoying to deal with.
Furthermore, the notification system is ancient.

## How
**Disclaimer: use at your own peril and please do not abuse it.**

#### Setup:
There are two files in the `docs` folder:
- `user.json`: contains the user's unique url number* and credentials.
- `classes.json`: contains the class' unique url number* and semester (1 or 2 for summer or winter, respectively).

*Just look for it in the url.

You must set **all** data on `user.json` and at leat one class on `classes.json`.

#### Running:
There are pre-built binaries in `bin`. Alternatively just run `go build .`, if you have it installed.
```
# class name as defined in the json file and the year of the files you wish to download (ex.: 22 for 2022)
./<binary-name> <class-name> <year>
```

## Extra
#### Why no concurrency?
I tested with various concurrent set-ups but it appears there is quite a strict rate-limit and I'm really not confident in CLIP's ability to not crash, even then. Don't wanna get my ass busted lmao.

#### Why do I have to fill everything myself?
It's just a one time setup but I agree; in the future I should query the database or something.