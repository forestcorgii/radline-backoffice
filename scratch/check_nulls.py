import sqlite3

conn = sqlite3.connect('backoffice.db')
cursor = conn.cursor()

cols = [c[1] for c in cursor.execute("PRAGMA table_info(sales_details)").fetchall()]
for col in cols:
    cnt = cursor.execute(f"SELECT COUNT(*) FROM sales_details WHERE [{col}] IS NULL").fetchone()[0]
    if cnt > 0:
        print(f"Column '{col}' has {cnt} NULL rows")

conn.close()
