# Raspberry Pi BLE Advertisement Setup

To broadcast BLE advertisements, the Raspberry Pi needs the Bluetooth radio unblocked and the BlueZ daemon running in "Experimental" mode (which enables newer Low Energy APIs).

## 1. Unblock Bluetooth (RF-Kill)

First, ensure the Bluetooth radio is not being blocked by the software kill switch.

    rfkill list all

If "Soft blocked" is set to yes, unblock it:

    sudo rfkill unblock bluetooth

## 2. Enable Experimental Features

By default, some BLE advertising features in the BlueZ stack are hidden behind an experimental flag.

Open the Bluetooth systemd service file:

    sudo nano /lib/systemd/system/bluetooth.service

(Note: On some systems, this might be located at /etc/systemd/system/bluetooth.target.wants/bluetooth.service)

Find the line starting with ExecStart. It usually looks like this: ExecStart=/usr/lib/bluetooth/bluetoothd

Add the --experimental (or -E) flag to the end of that line:

    ExecStart=/usr/lib/bluetooth/bluetoothd --experimental

    Save and exit (Ctrl+O, Enter, Ctrl+X).

## 3. Reload and Restart

Apply the changes by reloading the systemd daemon and restarting the Bluetooth service.

    sudo systemctl daemon-reload
    sudo systemctl restart bluetooth

## 4. Verification

Check that the service is running with the new flag and that the device is up.

    systemctl status bluetooth

Look for the ExecStart line in the log output to confirm it includes --experimental.

    hciconfig

You should see UP RUNNING.
