# Tubefling
Tubefling is a MVP to convert a Youtube channel to a MP3 podcast channel.
It **DOESN'T** require any Youtube API keys, and it only gives you the last 15 or so episodes, but it does what I want simply and easily.

To deploy use the Dockerfile, or, if you dont want that complexity, ensure you have the following:
* Python Modules
    * gunicorn
    * flask
    * requests 
    * pagan 
    * youtube-dl
* System packages
    * ffmpeg

You also probably want to throw gunicorn behind nginx, but [exoframe](https://github.com/exoframejs/exoframe) does that for me.

Tubefling respects two env vars:
* `TUBE_DIR`: Where to store downloaded/transcoded/generated files
* `AUTHORIZED_KEYS`: A comma seperated list of keys to use for auth

The URL format to use is: `https://:AUTHORIZED_KEY@HOSTING_URL/channel/CHANNEL_ID.xml`
* Yes you need the : before `AUTHORIZED_KEYS`
* CHANNEL_ID can be found on the page for the channel, either in the URL or the HTML, search and you'll find it

Note:
* Yes my server address is in deployment config, please don't nuke it
* You may need a nice long gunicorn timeout so transcodes don't fail, depending on how long your average video is
