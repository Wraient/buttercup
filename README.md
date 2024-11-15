
# buttercup

A cli application to stream torrents and track the playback using [jackett](https://github.com/Jackett/Jackett) and [webtorrent-cli](https://github.com/webtorrent/webtorrent-cli)

## Join the discord server

https://discord.gg/cNaNVEE3B6

## Demo

The demo speed has been increased, real speed would be slower depending on number of seeders and leechers

Cli Mode:

https://github.com/user-attachments/assets/59a17262-46b8-4cdd-bd79-3c362d72f2c6

Rofi Mode:

https://github.com/user-attachments/assets/c2c18047-b76a-4c98-a00d-fdab9e8fde5c


## Features
- Search for torrents
- Stream torrents
- Track playback
- Save MPV Speed
- Download / Install Jackett
- Config file
- CLI Selection menu
- Rofi Support

## Installing and Setup
> **Note**: `buttercup` requires `mpv`, `rofi`, and `webtorrent-cli` for Rofi support and torrent streaming. These are included in the installation instructions below for each distribution.

### Linux
<details>
<summary>Arch Linux / Manjaro (AUR-based systems)</summary>

Using Yay:

```bash
yay -Sy buttercup
```

or using Paru:

```bash
paru -Sy buttercup
```

Or, to manually clone and install:

```bash
git clone https://aur.archlinux.org/buttercup.git
cd buttercup
makepkg -si
sudo pacman -S rofi npm
npm install -g webtorrent-cli
```
</details>

<details>
<summary>Debian / Ubuntu (and derivatives)</summary>

```bash
sudo apt update
sudo apt install -y mpv curl rofi npm
sudo npm install -g webtorrent-cli
curl -Lo buttercup https://github.com/Wraient/buttercup/releases/latest/download/buttercup
chmod +x buttercup
sudo mv buttercup /usr/local/bin/
buttercup
```
</details>

<details>
<summary>Fedora Installation</summary>

```bash
sudo dnf update
sudo dnf install -y mpv curl rofi npm
sudo npm install -g webtorrent-cli
curl -Lo buttercup https://github.com/Wraient/buttercup/releases/latest/download/buttercup
chmod +x buttercup
sudo mv buttercup /usr/local/bin/
buttercup
```
</details>

<details>
<summary>openSUSE Installation</summary>

```bash
sudo zypper refresh
sudo zypper install -y mpv curl rofi npm
sudo npm install -g webtorrent-cli
curl -Lo buttercup https://github.com/Wraient/buttercup/releases/latest/download/buttercup
chmod +x buttercup
sudo mv buttercup /usr/local/bin/
buttercup
```
</details>

<details>
<summary>Generic Installation</summary>

```bash
# Install mpv, curl, rofi, npm, and webtorrent-cli (required for torrent streaming)
# Install npm for any additional packages

curl -Lo buttercup https://github.com/Wraient/buttercup/releases/latest/download/buttercup
chmod +x buttercup
sudo mv buttercup /usr/local/bin/
buttercup
```
</details>

<details>
<summary>Uninstallation</summary>

```bash
sudo rm /usr/local/bin/buttercup
```

For AUR-based distributions:

```bash
yay -R buttercup
```
</details>

## Usage

Run `buttercup` with the following options:

```bash
buttercup [options]
```

### Options



### Examples

- **Play with Rofi**:
  ```bash
  buttercup -rofi
  ```

## Configuration

All configurations are stored in a file you can edit with the `-e` option.

```bash
buttercup -e
```

Script is made in a way that you use it for one session of watching.

You can quit it anytime and the resume time would be saved in the history file

more settings can be found at config file.
config file is located at ```~/.config/buttercup/config```

## Dependencies
- mpv - Video player (vlc support might be added later)
- rofi - Selection menu
- tar - Download and unzip Jackett

## API Used
- [Jackett](https://github.com/Jackett/Jackett) - To get torrents
