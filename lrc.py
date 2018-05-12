import re
import requests
import sys
import time
import urllib
from bs4 import BeautifulSoup

search_url = 'https://mojim.com/song_name.html?t3'
lrc_url = 'https://mojim.com/twthxsong_idx1.htm'
singer = ''
song_list = []
song_file = 'songs.txt'
lrc_file = 'lrc.txt'

log_file = open('log.txt', 'a+', encoding='utf-8', newline='\n')

ENABLE_PROXY=1
if ENABLE_PROXY:
    proxy = {
        'http': 'http://127.0.0.1:8118',
        'https': 'https://127.0.0.1:8118'
    }
else:
    proxy = {}

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
    params = len(sys.argv)
    if params >= 4:
        song_file = sys.argv[2]
        lrc_file = sys.argv[3]
    elif params == 3:
        song_file = sys.argv[2]
    elif params < 2:
        print('Error')

    f = open(song_file, 'r', encoding="utf-8")
    line = f.readline()
    while line:
        song_list.append(line.strip('\n'))
        line = f.readline()
    f.close()

    singer = sys.argv[1]

    f = open(lrc_file, 'w+', encoding='utf-8', newline='\n')
    for song_name in song_list:
        org_name = song_name
        song_name = song_name.replace('-', ' ')
        song_id = get_song_id(song_name)
        if song_id != None:
            lrc = get_song_lrc(song_id)
            times = time.strftime('%Y-%m-%d %H:%M:%S', time.localtime())
            log = times + ' Fetching LRC: %s\n' % (org_name)
            log_file.write(log)
            f.write(lrc)
            print(log)
        else:
            times = time.strftime('%Y-%m-%d %H:%M:%S', time.localtime())
            log = times + ' No LRC: %s\n' % (org_name)
            log_file.write(log)
            print(log)
    f.close()
    log_file.close()
