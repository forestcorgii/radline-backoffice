import sqlite3

conn = sqlite3.connect('backoffice.db')
cursor = conn.cursor()

cursor.execute("SELECT COUNT(*) FROM receiving_logs WHERE supplier IS NULL")
print("Null suppliers count:", cursor.fetchone()[0])

conn.close()
