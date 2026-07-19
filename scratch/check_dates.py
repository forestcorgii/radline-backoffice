import sqlite3

conn = sqlite3.connect('backoffice.db')
cursor = conn.cursor()

cursor.execute("SELECT id, date FROM receiving_logs WHERE date IS NULL OR date = '' OR length(date) < 10")
bad_dates = cursor.fetchall()
print("Bad dates count:", len(bad_dates))
if bad_dates:
    print("Sample bad dates:", bad_dates[:10])

cursor.execute("SELECT DISTINCT date FROM receiving_logs LIMIT 20")
print("Sample distinct dates:", cursor.fetchall())

conn.close()
