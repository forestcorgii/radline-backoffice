import urllib.request
import urllib.parse
import http.cookiejar

cj = http.cookiejar.CookieJar()
opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj))

# Login
login_data = urllib.parse.urlencode({'username': 'admin', 'password': 'admin123'}).encode('utf-8')
req = urllib.request.Request('http://localhost:8080/login', data=login_data, headers={'HX-Request': 'true'})
resp = opener.open(req)
print("Login status:", resp.status)

# Fetch dashboard
req2 = urllib.request.Request('http://localhost:8080/')
resp2 = opener.open(req2)
content = resp2.read().decode('utf-8')

with open('scratch/dashboard_rendered.html', 'w', encoding='utf-8') as f:
    f.write(content)

print("Saved dashboard_rendered.html. Length:", len(content))
print("Goals link has target:", 'href="/goals" hx-get="/goals" hx-target="#main-content"' in content)
print("Reminders link has target:", 'href="/reminders" hx-get="/reminders" hx-target="#main-content"' in content)
print("Goals card present:", 'Earning Goals' in content)
print("Reminders card present:", 'Upcoming Reminders' in content)

