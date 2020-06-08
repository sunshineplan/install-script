#!/usr/bin/env python3

from csv import DictReader
from io import BytesIO
from subprocess import Popen, check_call, check_output
from time import sleep, time
from zipfile import ZipFile

import requests
from click import pause

proxies = {'https': 'http://localhost:1080'}

api = 'https://api.github.com/repos/v2fly/v2ray-core/releases/latest'


def getDownloadURL():
    response = requests.get(api, proxies=proxies, timeout=10).json()
    files = [i['browser_download_url'] for i in response['assets']]
    for f in files:
        if 'windows-64' in f and 'dgst' not in f:
            target = f
            return target, response['tag_name']


def checkVersion(new):
    current = 'v' + check_output('v2ray.exe -version').decode().split(' ')[1]
    if current == new:
        return 0
    return 1  # New version found.


def download(target):
    download_list = ['geoip.dat', 'geosite.dat',
                     'v2ctl.exe',  'v2ray.exe', 'wv2ray.exe']
    start = time()
    response = requests.get(target, proxies=proxies, timeout=10, stream=True)
    total = int(response.headers['content-length'])
    downloaded = 0
    zip = BytesIO()
    print(f"[{' '*50}]   0.0 KB/s -   0.0% of {total/1048576:.2f}M", end='\r')
    for chunk in response.iter_content(1024*512):
        if chunk:
            downloaded += len(chunk)
            zip.write(chunk)
            done = int(50 * downloaded / total)
            speed = downloaded/(time()-start)/1024
            left = (total-downloaded)/(speed*1024)
            if speed >= 1024:
                speed = f'{speed/1024:.1f} MB/s'
            else:
                speed = f'{speed:.1f} KB/s'
            if left >= 60:
                left = f'{left/60:.1f} mins left'
            else:
                left = f'{left:.0f} secs left'
            print(
                f"[{'='*done}{' '*(50-done)}] {speed:>10} - {downloaded/total:>6.1%} of {total/1048576:.2f}M, {left:<14}", end='\r')
    if downloaded != total:
        raise Warning('Download failed.')
    print(f"[{'='*done}{' '*(50-done)}] {speed:>10} - {downloaded/total:>6.1%} of {total/1048576:.2f}M, {left:<14}")
    print(f'Download finished. Total time: {time()-start:.2f}s')
    image, running = shutdown()
    with ZipFile(zip) as zf:
        for i in download_list:
            with open(i, 'wb') as f:
                f.write(zf.read(i))
    return image, running


def shutdown():
    tasks = DictReader(check_output('tasklist /fo csv').decode().splitlines())
    for t in tasks:
        if 'v2ray' in t['Image Name']:
            print('Shutting down V2Ray service.')
            check_call('taskkill /f /pid '+t['PID'])
            sleep(1)
            return t['Image Name'], 1
    return '', 0


def main():
    try:
        target, tag_name = getDownloadURL()
    except:
        print('Failed to get download URL.')
        return
    if checkVersion(tag_name):
        print('New version found.')
        print(f'Downloading {tag_name}')
        try:
            image, running = download(target)
        except Exception as e:
            print(str(e))
            return
        if running:
            print('Restarting V2Ray service.')
            Popen(image, creationflags=16)  # CREATE_NEW_CONSOLE
    else:
        print(f'Latest version {tag_name} is already installed.')


if __name__ == '__main__':
    main()
    pause()
