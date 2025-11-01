"""
Diagnostic test to understand what state the application reaches
"""

import pytest
import time
import requests


@pytest.mark.timeout(300)
def test_monitor_application_state(smart_crop_container):
    """Monitor what states the application goes through"""
    base_url = smart_crop_container["base_url"]

    print("\n=== Monitoring Application State ===")

    for i in range(60):  # Monitor for up to 2 minutes
        try:
            response = requests.get(f"{base_url}/api/status", timeout=10)
            data = response.json()

            print(f"\n[{i*2}s] Status: {data.get('status')}")
            print(f"  Progress: {data.get('progress')}%")
            print(f"  Message: {data.get('message')}")
            print(f"  Candidates: {len(data.get('candidates', []))}")
            print(f"  Selected: {data.get('selected_index')}")
            print(f"  Acceleration: {data.get('enable_acceleration')}")

            # Check if we're stuck waiting for input
            if data.get('status') in ['awaiting_selection', 'candidates_ready', 'awaiting_acceleration_choice']:
                print(f"\n⚠️  Application is waiting for user input at state: {data.get('status')}")
                print("  This explains why analysis doesn't 'complete' - it's waiting for interaction")
                break

        except Exception as e:
            print(f"\n[{i*2}s] Error: {e}")
            break

        time.sleep(2)

    print("\n=== End Monitoring ===")


if __name__ == "__main__":
    pytest.main([__file__, "-v", "-s"])
