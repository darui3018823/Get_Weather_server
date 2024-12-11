import requests

# APIのエンドポイントURL
api_url = 'http://localhost:8080/weather'

# リクエストデータ
request_data = {
    "program_type": "Get_Weather",
    "program_langs": "Golang",
    "data": {
        "prefname": "東京都",
        "cityname": "東京"
    }
}

# POSTリクエストを送信
headers = {'Content-Type': 'application/json'}
response = requests.post(api_url, json=request_data, headers=headers)

# レスポンスの処理
if response.status_code == 200:
    print("Request was successful.")
    print("Response Data:", response.json())
else:
    print(f"Request failed with status code: {response.status_code}")
    print("Error:", response.text)
