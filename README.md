# Util
This is my utility repo. The original motivation for this is that when I change jobs I lose the little utilities I've put together for myself and that's annoying. A former colleague, [Michael Jarvis](https://www.linkedin.com/in/michaeljarvis/) shared the idea of keeping one's shell environment in a public repository so that wherever you setup you bring your utils and environment with you. This repo doesn't go quite that far, but is a step in that direction.


## Installation
Prerequisites:
  * Docker needs to be installed and accessible as `docker`

Setup:
  * Add the entries in [profile](profile) to your shell profile, such as `.bashrc` or `.zshrc` or wherever you want.
    * You can clone the repo and then run `cat profile >> .bashrc` or something like that
    * You can literally just open [profile](profile) and paste into your relevant file; you don't need anything else from this repo

## Inventory
The only things I actually have in here right now are image and video conversion tools. This is becuase I often need to share content in various ways that are made easier with some conversion.
### heic_to_jpeg
I use Apple products so photos are often taken in HEIC, but a lot of tools and folks can only consume non-Apple formats. This just converts an image with a oneliner.
### reduce_framerate
Given a .mov file (e.g. a Quicktime screen recording), simply reduce its size by reducing the framerate. Not suitable for everything, but most screen recordings I'm trying to sling to a colleague don't need full framerate and this is a very easy way to SIGNIFICANTLY reduce file size.
### mov_to_gif
Converts a given .mov file to an animated GIF, such as to include in a repository like the demo GIF.

## Demo
![demo.gif](demo.gif)
## Security
You're somewhat trusting [this Docker image proffered via DockerHub](https://hub.docker.com/repository/docker/nickborgers/mov-to-gif/general). It's built from this repo in a [GitHub Workflow](.github/workflows/publish.yml). My GitHub account is protected with FIDO/FIDO2 (only possible to establish a session with FIDO/FIDO2) and I don't store my SSH key in cleartext on any disk. GitHub has a DockerHub token and the DockerHub account password is stored only in an offline password database.

Note that the Docker containers are given no network interface (`--network=none`), so I see no way the image could exfil your data even if it wanted to.
