import sqlite3

conn = sqlite3.connect('backoffice.db')
cursor = conn.cursor()

cursor.execute("SELECT id, supplier, date, pl_no, unit_price, selling_price FROM receiving_logs WHERE pl_no IS NULL OR unit_price IS NULL OR selling_price IS NULL LIMIT 5")
null_rows = cursor.fetchall()
print("Null rows count/sample:", len(null_rows), null_rows)

conn.close()
