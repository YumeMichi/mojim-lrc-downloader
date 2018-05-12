import re
import requests
import urllib
from bs4 import BeautifulSoup

search_url = 'https://mojim.com/song_name.html?t3'
lrc_url = 'https://mojim.com/twthxsong_idx1.htm'
singer = 'FictionJunction'
song_list = [
    'circus',
    'aikoi',
    'Silly-Go-Round',
    '焔の扉',
    'よろこび',
    '荒野流転',
    'ピアノ',
    '暁の車',
    'cazador del amor',
    '記憶の森',
    'nowhere',
    '約束'
]

proxy = {
    'http': 'http://127.0.0.1:8118',
    'https': 'https://127.0.0.1:8118'
}

def get_song_id(song_name):
    req = requests.get(search_url.replace('song_name', song_name), proxies=proxy)
    data = req.text

    soup = BeautifulSoup(data, 'lxml')
    spans = soup.findAll('span', {
        'class': 'mxsh_ss4'
    })

    patt = re.compile(r"(.*?) " + singer)

    for sp in spans:
        a = sp.find('a', {
            'title': patt
        })
        if a != None:
            return a.attrs['href'].replace('/twy', '').replace('.htm', '')

    return None

def get_song_lrc(song_id):
    req = requests.get(lrc_url.replace('song_id', song_id), proxies=proxy)
    data = req.text

    soup = BeautifulSoup(data, "html.parser")
    patt = re.compile(r"var swfmm = \"(.*?)\";")
    scrp = soup.find("script", text=patt)

    lrc = patt.search(scrp.text).group(1).replace("_", "%")
    dec = urllib.parse.unquote(lrc)

    return dec

if __name__ == '__main__':
    for song_name in song_list:
        song_name = song_name.replace('-', ' ')
        song_id = get_song_id(song_name)
        if song_id != None:
            lrc = get_song_lrc(song_id)
            print(lrc)
        else:
            print('No results for song: %s!' % (song_name))