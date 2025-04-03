from bs4 import BeautifulSoup
import requests, os, re, json, sys
from tqdm import tqdm

def download_video(url, cookie, save_path):
    try:
        headers_cookies = {
            "accept": "*/*",
            "accept-encoding": 'identity;q=1, *;q=0',
            "accept-language": 'zh-TW,zh;q=0.9,en-US;q=0.8,en;q=0.7',
            "cookie": cookie,
            "dnt": '1',
            "user-agent": 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3945.130 Safari/537.36'
        }
        # 发起 HTTP GET 请求
        response = requests.get(url, headers = headers_cookies, stream=True)
        response.raise_for_status()

        total_size = int(response.headers.get('content-length', 0))

        with open(save_path, 'wb') as f, tqdm(
                desc=save_path,
                total=total_size,
                unit='B',
                unit_scale=True,
                unit_divisor=1024,
        ) as bar:
            for chunk in response.iter_content(chunk_size=8192):
                f.write(chunk)
                bar.update(len(chunk))

        #print(f"\n视频已保存到: {save_path}")
    except Exception as e:
        print(f"下载失败: {e}")

def getVideoByURL(url, save_base_path):

    if not os.path.exists(save_base_path):
        os.mkdir(save_base_path)

    headers = {
        "Accept": "*/*",
        "Accept-Language": 'zh-TW,zh;q=0.9,en-US;q=0.8,en;q=0.7',
        "DNT": "1",
        "Sec-Fetch-Mode": "cors",
        "Sec-Fetch-Site": "same-origin",
        "cookie": "__cfduid=d8db8ce8747b090ff3601ac6d9d22fb951579718376; _ga=GA1.2.1940993661.1579718377; _gid=GA1.2.1806075473.1579718377; _ga=GA1.3.1940993661.1579718377; _gid=GA1.3.1806075473.1579718377",
        "Content-Type":"application/x-www-form-urlencoded",
        "user-agent": "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/71.0.3573.0 Safari/537.36",
    }

    r = requests.post(url, headers = headers)

    soup = BeautifulSoup(r.text, 'lxml')
    videos = soup.find_all('video', class_='video-js')
    results = [video['data-apireq'] for video in videos if video.has_attr('data-apireq')]
    title = soup.find('h2', class_="entry-title").text
    for rslt in results:
        xsend = 'd={}'.format(rslt)
        r = requests.post('https://v.anime1.me/api',headers = headers,data = xsend)

        set_cookie = r.headers['set-cookie']
        cookie_e = re.search(r"e=(.*?);", set_cookie, re.M|re.I).group(1)
        cookie_p = re.search(r"p=(.*?);", set_cookie, re.M|re.I).group(1)
        cookie_h = re.search(r"HttpOnly, h=(.*?);", set_cookie, re.M|re.I).group(1)
        cookies = 'e={};p={};h={};'.format(cookie_e, cookie_p, cookie_h)


        srcInfo = json.loads(r.text)
        srcUrl = "http:" + srcInfo["s"][0]["src"]
        #print(srcUrl)
        nm =  os.path.basename(srcUrl).replace("b.", ".")
        fnm = save_base_path + "\\" + nm
        print("开始下载：" + nm)
        download_video(srcUrl, cookies, fnm)
        print(nm + "下载完毕")

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("用法：输入anime1.me的动漫界面的链接，然后输入保存的文件夹名")
        sys.exit(0)
    url = sys.argv[1]
    save_base_path = sys.argv[2]
    getVideoByURL(url, save_base_path)