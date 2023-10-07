# Running this server via docker-osx
> WIP, not yet end to end tested

This guide expects the host to have a x11 session.
But it can probable be setup headless using the VNC options from [docker-osx](https://github.com/sickcodes/Docker-OSX#building-a-headless-container-that-allows-insecure-vnc-on-localhost-for-local-use-only)

## Follow the [initial setup](https://github.com/sickcodes/Docker-OSX#initial-setup)

## Run ... to start docker-osx to install ventura
```bash
  # to capture and reuse hardware ids
  touch output.env
  
  docker run -it \
      --device /dev/kvm \
      -p 50922:10022 \
      -v /tmp/.X11-unix:/tmp/.X11-unix \
      -v ./output.env:/env \
      -e "DISPLAY=${DISPLAY:-:0.0}" \
      -e GENERATE_UNIQUE=true \
      -e MASTER_PLIST_URL='https://raw.githubusercontent.com/sickcodes/osx-serial-generator/master/config-custom.plist' \
      sickcodes/docker-osx:ventura
      # optional
      # if you experience network problems try adding
      # --net host \
      # also you can change the resolution with
      # -e WIDTH=1600 \
      # -e HEIGHT=900 \
  
```
If you encounter any problems check the docker-osx repo's [issues](https://github.com/sickcodes/Docker-OSX/issues)

## When OSX is running follow the these steps

- Double click `Disk Utility`
- Select the `QEMU HARDDISK Media` with ~ 280GB
- Click on `Erase` and then again `Erase`
- When done close the `Disk Utility`
- Double click `Reinstall ventura`
- Follow the installation
- Create a user
- Login
- [Enable ssh](https://support.apple.com/guide/mac-help/allow-a-remote-computer-to-access-your-mac-mchlp1066/mac)
- shutdown ventura

## Run ... to find the latest  `mac_hdd_ng.img` created
```bash
sudo bash -c "find /var/lib/docker -name mac_hdd_ng.img -type f -print0 | xargs -0 ls -lt | head -1"
```

Copy the image to your local directory

## complete the mac address
the mac address in the `output.env` is incomplete you have to prefix it with one of apples. (see: https://www.adminsub.net/mac-address-finder/apple)

## create docker-compose.yml and customize to your needs
```yaml
version: "3.9"

services:
  macos:
    image: sickcodes/docker-osx:naked
    # for later: image: sickcodes/docker-osx:naked-auto
    devices:
      - /dev/kvm
    ports:
      - 127.0.01:50922:10022
    volumes:
      - /tmp/.X11-unix:/tmp/.X11-unix 
      - ./mac_hdd_ng.img:/image
    env_file:
      - ./output.env
    environment:
      USERNAME: <username>
      PASSWORD: <password>
      DISPLAY: "${DISPLAY:-:0.0}"
      # for later OSX_COMMANDS: "/bin/bash /Users/<username>/shortcut-signing-server/docker-osx/start-server.sh"
      GENERATE_SPECIFIC: true
      # DEVICE_MODEL: "iMacPro1,1"
      # WIDTH: "1600"
      # HEIGHT: "900"
      # for later: ADDITIONAL_PORTS: "hostfwd=tcp::10023-:80,"
```

## setup the server for auto-run
- run `docker compose up -d`
- log in via ui or via ssh `ssh <user>@127.0.0.1 -p 50922`
- (option 1) install golang and git manual
- (option 2) install nix
  - run `sh <(curl -L https://nixos.org/nix/install)`
  - after installation run `nix-env -iA nixpkgs.go -iA nixpkgs.git`
- run `git clone https://github.com/scaxyz/shortcut-signing-server`
- run `cd shortcut-signing-server && go install .`
- run `cd docker-osx`
- create a config file `config.yaml`
- customize the config
- make sure you are logged into your (burner?) icloud account
  - (needed but only possible via ui) 
- shutdown the container with `docker compose stop`
- modify your `docker-compose.yml`
  - change the image to `sickcodes/docker-osx:naked-auto`
  - add `OSX_COMMANDS: /bin/bash /Users/<username>/shortcut-signing-server/docker-osx/start-server.sh` to the `environment` section
  - add `ADDITIONAL_PORTS: "hostfwd=tcp::10023-:80,"` to the `environment` section
  - add `80:10023` or `443:10023` to the `ports` section
- run `docker compose up` and check if everything is working
  - `curl http://localhost:<PORT>` should result in error 404
  - visiting `http://localhost:\<PORT>/sign` should show a simple form for signing shortcuts
- ~~if everything is ok, comment out  `/tmp/.X11-unix:/tmp/.X11-unix ` in your docker-compose.yml to disable/hide the ui~~
  - currently you need to unlock your machine via the ui/vnc to be able to sign shortcuts
- copy your docker-compose file and image to your server if not already there
