#!/usr/bin/env python3

from csv import DictReader
from io import BytesIO
from subprocess import Popen, check_call, check_output
from time import time

import requests
from click import pause

proxies = {'https': 'http://localhost:1080'}

api = 'https://api.github.com/repos/geph-official/gephng-binaries/commits'


def getDownloadURL():
    sha = requests.get(api, proxies=proxies, timeout=10).json()[0]['sha']
    files = [f['raw_url']
             for f in requests.get(api+'/'+sha, proxies=proxies, timeout=10).json()['files']]
    for f in files:
        if 'geph-client-windows' in f:
            target = f
            return target


def checkVersion(new):
    try:
        with open('version', 'r') as f:
            current = f.read().strip()
    except FileNotFoundError:
        print('Check version failed.')
        return 1
    if current == new:
        print(f'Latest version v{new} is already installed.')
        return 0
    else:
        print('New version found.')
        return 1


def download(target):
    start = time()
    response = requests.get(target, proxies=proxies, timeout=10, stream=True)
    total = int(response.headers['content-length'])
    downloaded = 0
    content = BytesIO()
    print(f"[{' '*50}]   0.0 KB/s -   0.0% of {total/1048576:.2f}M", end='\r')
    for chunk in response.iter_content(1024*512):
        if chunk:
            downloaded += len(chunk)
            content.write(chunk)
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
    running = shutdown()
    try:
        with open('geph.exe', 'wb') as f:
            f.write(content.getvalue())
    except PermissionError:
        with open('geph.tmp', 'wb') as f:
            f.write(content.getvalue())
        raise Warning('Failed to update geph.exe file.')
    return running


def shutdown():
    tasks = DictReader(check_output('tasklist /fo csv').decode().splitlines())
    for t in tasks:
        if 'geph' in t['Image Name']:
            print('Shutting down Geph service.')
            check_call('taskkill /f /pid '+t['PID'])
            return 1
    return 0


def main():
    try:
        target = getDownloadURL()
    except:
        print('Failed to get download URL.')
        return
    tag_name = target.split('-v')[1].replace('.exe', '')
    if checkVersion(tag_name):
        print(f'Downloading v{tag_name}')
        try:
            running = download(target)
        except Exception as e:
            print(str(e))
            return
        with open('version', 'w') as f:
            f.write(tag_name)
        if running:
            print('Restarting Geph service.')
            Popen('geph.exe -config client.conf',
                  creationflags=134217728)  # CREATE_NO_WINDOW


if __name__ == '__main__':
    main()
    pause()
