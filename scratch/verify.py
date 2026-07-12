import urllib.request
import urllib.parse
import sys

# Force stdout/stderr to use UTF-8 to prevent cp1252 charmap errors on Windows
if hasattr(sys.stdout, 'reconfigure'):
    sys.stdout.reconfigure(encoding='utf-8')
if hasattr(sys.stderr, 'reconfigure'):
    sys.stderr.reconfigure(encoding='utf-8')

def test_endpoint(name, url, method="GET", data=None, expected_texts=[]):
    print(f"Testing {name} ({method} {url})...")
    req = urllib.request.Request(url, method=method)
    if data:
        encoded_data = urllib.parse.urlencode(data).encode("utf-8")
        req.data = encoded_data
        req.add_header("Content-Type", "application/x-www-form-urlencoded")
    
    # Simulate HTMX requests for fragments
    if method in ["POST", "DELETE"] or "tab=" in url:
        req.add_header("HX-Request", "true")
        
    try:
        with urllib.request.urlopen(req) as response:
            html = response.read().decode("utf-8")
            status = response.status
            print(f"  Response Status: {status}")
            
            # Check for expected substrings in response
            for text in expected_texts:
                if text not in html:
                    print(f"  [ERROR] Expected text '{text}' not found in response!")
                    # Print preview of response for debugging
                    print(f"  Response Preview: {html[:300]}")
                    return False
            print("  [SUCCESS] All checks passed.")
            return True
    except Exception as e:
        print(f"  [ERROR] Request failed: {e}")
        return False

def main():
    # Clean up test data from previous runs to ensure reproducibility
    try:
        import sqlite3
        conn = sqlite3.connect("backoffice.db")
        cursor = conn.cursor()
        cursor.execute("DELETE FROM sales_details WHERE item_id IN (SELECT id FROM items WHERE code='IT1')")
        cursor.execute("DELETE FROM receiving_logs WHERE item_id IN (SELECT id FROM items WHERE code='IT1')")
        cursor.execute("DELETE FROM inventory_adjustments WHERE item_id IN (SELECT id FROM items WHERE code='IT1')")
        cursor.execute("DELETE FROM items WHERE code='IT1'")
        conn.commit()
        conn.close()
        print("Cleaned up previous test data successfully.")
    except Exception as e:
        print(f"Warning: Cleanup failed: {e}")

    base_url = "http://localhost:8080"
    success = True

    # 1. Test Items view
    success &= test_endpoint("GET Items Page", f"{base_url}/items", expected_texts=["Items List"])

    # 1b. Test Entry Page (Item tab)
    success &= test_endpoint("GET Entry Page (Item)", f"{base_url}/entry?tab=item", expected_texts=["Add Item"])

    # 2. Add an Item
    item_data = {
        "code": "IT1",
        "description": "Test Product Item",
        "model": "M1",
        "brand_id": "1",
        "category_id": "1",
        "default_uom": "PCS"
    }
    success &= test_endpoint("Add Item", f"{base_url}/items/add", "POST", item_data, ["IT1", "Test Product Item", "M1", "PCS"])

    # Query the newly inserted item ID dynamically to support auto-incrementing IDs
    item_id = "1"
    try:
        import sqlite3
        conn = sqlite3.connect("backoffice.db")
        cursor = conn.cursor()
        cursor.execute("SELECT id FROM items WHERE code='IT1'")
        row = cursor.fetchone()
        if row:
            item_id = str(row[0])
        conn.close()
        print(f"Dynamically resolved item ID for IT1: {item_id}")
    except Exception as e:
        print(f"Warning: Failed to query item ID: {e}")

    # 3. Receive stock for the newly added item
    receive_data = {
        "item_id": item_id,
        "supplier": "ASCD",
        "pl_no": "PL100",
        "date": "2026-07-08",
        "qty": "10",
        "uom": "PCS",
        "cost": "100.0",
        "unit_price": "150.0"
    }
    success &= test_endpoint("Receive Stock", f"{base_url}/inventory/receiving/add", "POST", receive_data, ["PL100", "ASCD", "+10", "PCS", "₱150.00", "₱1000.00"])

    # 4. Make an Inventory Adjustment (subtracting 2 items)
    adjustment_data = {
        "item_id": item_id,
        "adjustment_qty": "-2",
        "uom": "PCS",
        "cost": "100.0",
        "remarks": "Damaged count adjustment"
    }
    success &= test_endpoint("Adjust Stock", f"{base_url}/inventory/adjustments/add", "POST", adjustment_data, ["-2", "PCS", "₱100.00", "Damaged count adjustment"])

    # 5. Encode a Sales Log (selling 3 items)
    sale_data = {
        "doc_date": "2026-07-08",
        "doc_type": "SI",
        "doc_number": "INV200",
        "customer_name": "John Doe",
        "supplier": "ASCD",
        "item_id": item_id,
        "qty": "3",
        "uom": "PCS",
        "price": "150.0",
        "cost": "100.0"
    }
    success &= test_endpoint("Encode Sales", f"{base_url}/sales/add", "POST", sale_data, ["-3", "INV200", "ASCD", "₱450.00", "₱150.00"])

    # 6. Verify Dashboard updates dynamically
    # Total Sales: 3 * 150.0 = 450.00
    # Total Costs: 3 * 100.0 = 300.00
    # Gross Profit: 450.0 - 300.0 = 150.00
    # Profit Margin: 150.0 / 450.0 = 33.3%
    success &= test_endpoint("GET Dashboard Page", f"{base_url}/", expected_texts=["Financial Overview", "₱450.00", "₱300.00", "₱150.00", "33.3%"])

    if success:
        print("\nAll automated integration tests passed successfully!")
        sys.exit(0)
    else:
        print("\nSome tests failed. See errors above.")
        sys.exit(1)

if __name__ == "__main__":
    main()
