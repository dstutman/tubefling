# Tubefling
Tubefling is a MVP to convert a Youtube channel to a MP3 podcast channel.
It DOESN'T require any Youtube API keys, but only gives you the last 15 or so episodes.

To deploy use the Dockerfile, or, if you dont want that complexity, ensure you have the following:
* Go > 1.12

Tubefling respects three env vars:
* `STATIC_DIR`: Where to store transcoded/generated files
* `TMP_DIR`: Where to store downloaded files
* `AUTHORIZED_KEYS`: A comma seperated list of keys to use for auth

The URL format to use is: `https://:AUTHORIZED_KEY@HOSTING_URL/channel/CHANNEL_ID.xml`
* Yes you need the : before `AUTHORIZED_KEYS`
* CHANNEL_ID can be found on the page for the channel, either in the URL or the HTML, search and you'll find it
