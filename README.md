## Config

<p>You have to modify the <u>config/constants.go</u> file and change source <b>ORIGIN_PATH</b> and <b>DEST_PATH</b> by your download folder and your wanted destination folder </p>

## Install Dependancies

```bash
make init
```

<details>

<summary>Or you can install dependencies by your own</summary>

<hr>
<b><u>mkvtoolnix:</u></b>

``` bash
sudo apt install mkvtoolnix
```

<b><u>Other Dependencies:</u></b>

``` bash
go mod tidy
go mod download
```
<hr>
</details>

#### Start

```bash
make start
```

#### Build

```bash
make build
```