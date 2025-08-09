# Comic-Hero

A community driven comic book reading order app

> [!WARNING]
> This app is currently in alpha. While it's usable, many features are missing.

## Free, in more ways than one

- You will never have to pay for this app. No feature will ever be hidden behind a paywall.
- The entire project is open source. You can create your own fork of this project, host your own version of this app and help improve it. 
- Not only is the project open source, the end goal of this project is to make reading orders community driven. Copy, edit and integrate reading orders instead of just looking at them.

## Searching for and adding a comic book

There are currently two ways of searching for a comic book. Searching the apps own database and searching the [Metron comic book database](https://metron.cloud).

Whenever a comic book from a Metron search is added to a reading order, this comic will be added to the the apps database. As of now this is also the only way to add a comic to the database.

> [!CAUTION]
> Currently there is no check whether a comic from Metron was already added to the apps database.
> In order to prevent duplicate comics, please search the apps own database first and do not add a Metron comic that already exists in the apps database.

> [!NOTE]
> If you can't find a comic, check if that comic is available on [Metron](https://metron.cloud/issue/). If it is not available, consider supporting Metron by adding the comic you are missing. When doing so, please respect [Metrons editing guidelines](https://metron.cloud/pages/guidelines/editing/)

## Roadmap

- [ ] Search and pagination for reading order
- [ ] Delete reading order from UI
- [ ] Edit reading order (name, description)
- [ ] Show entry notes in reading order list
- [ ] Copy reading order
- [ ] Add another reading order as a sub-list
- [ ] Add OAuth2 and MFA
- [ ] Pagination in comic search

## Technical

### Spin up an instance of ComicHero

All ComicHero compoents have a corresponding Dockerfile and a docker-compose file is available to build, configure and start all necessary services inside docker containers. 

The easiest way to run and manage the docker services is by cloning this repository using git and in the projects root folder run 

```sh
$ docker compose up --build
```

A configuration needs to be provided for the web application. For this, a binding in the docker compose file has been set up to `./config/`. This means that the directory `config` needs to exist in directory the compose command is being executed. Place a file `config.json` inside this folder with the following content:

```json
{
    "backendUrl": "<base url>:8090",
    "metronProxyUrl": "<base url>:8080"
}
```

The ports are configured in the compose file.

### General architecture

- The backend is a [PocketBase](https://pocketbase.io) instance. 
    - Easy API creation with batteries included
    - Comes with an admin UI to manage API as well as database and data
- The front end is written in [Flutter](https://flutter.dev)
    - One front end code base, multiple platforms
    - Web-Version available
    - (Desktop versions for Linux, Windows and MacOS planned)
    - (Native iOS and Android verions currently not available)
- A proxy for Metron written in [Go](https://go.dev)
    - A very simple proxy that will pass requests directly to Metron
    - The proxy will also cache Metron calls for all users, reducing stress to the Metron API
    - Due to Metron not having any way to enable CORS, this proxy is necessary to make Metron API calls available in the web-version