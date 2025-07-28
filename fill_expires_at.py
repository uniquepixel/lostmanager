import psycopg2

# Connection settings
# dsn = "postgres://lostapp:R0WBrAD0xWdGVpCvc0NM6Yd5KdE4QrUj@192.168.178.103:5435/lostapp"
dsn = "postgres://lostapp:R0WBrAD0xWdGVpCvc0NM6Yd5KdE4QrUj@45.81.234.14:5432/lostapp"


def main():
    # Connect to the database
    conn = psycopg2.connect(dsn)
    cur = conn.cursor()

    # Update expires_at for each kickpoints entry by adding the clan-specific expiration days
    update_sql = """
    UPDATE kickpoints AS kp
    SET expires_at = kp.date + (cs.kickpoints_expire_after_days * INTERVAL '1 day')
    FROM clan_settings AS cs
    WHERE kp.clan_tag = cs.clan_tag
      AND (
        kp.expires_at IS NULL
        OR kp.expires_at
           IS DISTINCT FROM kp.date + (cs.kickpoints_expire_after_days * INTERVAL '1 day')
      );
    """

    try:
        # Execute the update statement
        cur.execute(update_sql)
        # Prompt user before committing
        print(f"This will affect {cur.rowcount} rows. Commit changes? (y/n): ", end="")
        choice = input().strip().lower()
        if choice in ('y', 'yes'):
            conn.commit()
            print(f"Committed changes to {cur.rowcount} rows.")
        else:
            conn.rollback()
            print("Rolled back; no changes were committed.")
    except Exception as e:
        conn.rollback()
        print("Error updating expires_at:", e)
    finally:
        cur.close()
        conn.close()


if __name__ == '__main__':
    main()
