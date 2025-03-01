# Normalize Video

<p><b>Normalize Video</b> is a tool designed to standardize the titles of your videos (movies and series) by reorganizing files from a source folder to a destination folder.</p> 

<p>The goal is to automate this process through scheduled tasks, so you no longer need to perform it manually. TV series are automatically classified by series name and season, with the entire directory structure created on the fly. After processing, a summary table displays all extracted information (language, quality, file extension, etc.) along with the total number of movies/series processed.</p>

<p>If mkvtoolnix is installed, the program will automatically update the video title in the MKV file and set the default audio track and subtitles to French (forcing French if the audio track is in VF). This behavior can be customized as needed.</p>

## Config

<p>In the file <u>config/constants.go</u>, set the following constants:</p>
<ul>
    <li><b>ORIGIN_PATH</b> <code>/path/to/source/folder/</code></li>
    <li><b>DEST_PATH</b> <code>/path/to/destination/folder/</code></li>
</ul>

## Install Dependancies

```bash
make init
```

<details>

<summary>Or install dependencies by your own</summary>

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