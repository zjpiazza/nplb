# Repository Setup Instructions

1. Download the keyring file:
   ```bash
   sudo wget -O /usr/share/keyrings/nplb.gpg https://nplb.wastelandsystems.io/keyring.gpg
   ```

2. Add the repository:
   ```bash
   echo "deb [signed-by=/usr/share/keyrings/nplb.gpg] https://nplb.wastelandsystems.io stable main" | sudo tee /etc/apt/sources.list.d/nplb.list
   ```

3. Update apt:
   ```bash
   sudo apt update
   ```
