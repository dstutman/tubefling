"""
Written by Daniel Avishai Stutman, 2019-05-15
Copyright 2019 Daniel Avishai Stutman
Licenced under the DWTFYWPL
"""

from flask import Flask, request, Response, render_template_string, redirect, url_for
from functools import wraps
import requests
import xml.etree.ElementTree as xml_parser
import datetime
import pagan
import youtube_dl
import os

# Directory for temporary files
tmp_dir = os.environ["TMP_DIR"]
auth_tokens = os.environ["AUTHORIZED_TOKENS"].split(",")

server = Flask(__name__, static_folder=tmp_dir)

def check_auth(token):
    return token in auth_tokens


def authenticate():
    return Response(
        "Could not verify your access level for that URL.\n"
        "You have to login with proper credentials",
        401,
        {"WWW-Authenticate": 'Basic realm="Login Required"'},
    )


def authenticated(f):
    @wraps(f)
    def decorated(*args, **kwargs):
        auth = request.authorization
        if not auth or not check_auth(auth.password):
            return authenticate()
        return f(*args, **kwargs)

    return decorated


@server.route("/")
def index():
    return """
        <!DOCTYPE html>
        <html>
        <head>
            <meta charset="UTF-8">
            <title>Tubefling</title>
        </head>
        <body>
            <h1>Convert a Youtube channel to an MP3 podcast</h1>
            <h2>Usage</h2>
            <p>
                Add the following link to your podcast client:
                https://:AUTHORIZED_KEY@HOSTING_URL/channel/CHANNEL_ID.xml
                dont forget the : preceeding YOUR_AUTH_TOKEN. The token has to be passed as a basicauth password, not a username.
            </p>
        </body>
        </html>
    """


@server.route("/avatar/<uuid>.png")
@authenticated
def avatar(uuid):
    file_name = f"avatar_{uuid}.png"
    pagan.Avatar(uuid, pagan.SHA512).save(tmp_dir, file_name)
    return redirect(url_for('static', filename=file_name), code=301)


def get_channel_data(chan_id):
    chan_url = f"https://youtube.com/feeds/videos.xml?channel_id={chan_id}"
    resp = requests.get(chan_url)
    root = xml_parser.fromstring(resp.text)
    video_ids = []
    for child in root:
        if child.tag == "{http://www.w3.org/2005/Atom}entry":
            video_ids.append(
                {
                    "title": child[3].text,
                    "desc": child[8][3].text,
                    "thumb_url": child[8][2].attrib["url"],
                    "id": child[1].text,
                    "published_date": child[6].text,
                }
            )
    return {
        "title": root[3].text,
        "homepage_url": root[5].get("href"),
        "id": chan_id,
        "videos": video_ids,
    }


@server.route("/channel/<chan_id>.xml")
@authenticated
def channel(chan_id):
    chan_data = get_channel_data(chan_id)
    chan_xml_str = render_template_string(
        """
<rss xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd" xmlns:atom="http://www.w3.org/2005/Atom" xmlns:rawvoice="http://www.rawvoice.com/rawvoiceRssModule/" version="2.0">
            <channel>
                <title>{{chan_data.title}}</title>
                <link>{{chan_data.homepage_url}}</link>
                <image>
                    <url>{{server_base_url}}avatar/{{chan_data.id}}.png</url>
                    <title>{{chan_data.title}}</title>
                    <link>{{chan_data.homepage_url}}</link>
                </image>
                <language>en-us</language>
                <copyright>{{chan_data.title}}</copyright>
                <lastBuildDate>{{last_build_date}}</lastBuildDate>
                <itunes:image href="{{server_base_url}}avatar/{{chan_data.id}}.png"/>
                {% for video in chan_data.videos %}
                <item>
                    <title>{{video.title}}</title>
                    <description>{{video.desc}}</description>
                    <itunes:summary>{{video.desc}}</itunes:summary>
                    <itunes:image href="{{video.thumb_url}}"/>
                    <guid>{{server_base_url}}video/{{video.id}}.mp3</guid>
                    <link>{{server_base_url}}video/{{video.id}}.mp3</link>
                    <enclosure url="{{server_base_url}}video/{{video.id}}.mp3" type="audio/mpeg"/>
                    <pubDate>{{video.published_date}}</pubDate>
                </item>
                {% endfor %}
            </channel>
        </rss>
    """,
        chan_data=chan_data,
        last_build_date=str(datetime.datetime.now()),
        server_base_url=request.url_root,
    )
    return Response(chan_xml_str, mimetype="text/xml")


@server.route("/video/<video_id>.mp3", methods=['GET','POST'])
@authenticated
def video(video_id):
    if not os.path.isfile(f"{tmp_dir}/transcoded_{video_id}.mp3"):
        ytdl_opts = {
            "format": "bestaudio/best",
            "outtmpl": f"{tmp_dir}/transcoded_%(id)s.%(ext)s",
            "postprocessors": [
                {
                    "key": "FFmpegExtractAudio",
                    "preferredcodec": "mp3",
                    "preferredquality": "192",
                }
            ],
        }
        with youtube_dl.YoutubeDL(ytdl_opts) as ytdl:
            ytdl.download([f"https://www.youtube.com/watch?v={video_id}"])
    return redirect(url_for('static', filename=f"transcoded_{video_id}.mp3"), code=301)

if __name__ == "__main__":
    server.run()
