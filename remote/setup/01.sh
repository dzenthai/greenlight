set -eu

TIMEZONE=Europe/Berlin

USERNAME=greenlight

read -p "Enter password for greenlight DB user: " DB_PASSWORD

export LC_ALL=en_US.UTF-8

add-apt-repository --yes universe

apt update

timedatectl set-timezone ${TIMEZONE}
apt --yes install locales-all

useradd --create-home --shell "/bin/bash" --groups sudo "${USERNAME}"

passwd --delete "${USERNAME}"
chage --lastday 0 "${USERNAME}"

rsync --archive --chown=${USERNAME}:${USERNAME} /root/.ssh /home/${USERNAME}

ufw allow 22
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable

apt --yes install fail2ban

curl -L https://github.com/golang-migrate/migrate/releases/download/v4.14.1/migrate.linux-amd64.tar.gz | tar xvz
mv migrate.linux-amd64 /usr/local/bin/migrate

apt --yes install postgresql

sudo -i -u postgres psql -c "CREATE DATABASE greenlight"
sudo -i -u postgres psql -d greenlight -c "CREATE EXTENSION IF NOT EXISTS citext"
sudo -i -u postgres psql -d greenlight -c "CREATE ROLE greenlight WITH LOGIN PASSWORD '${DB_PASSWORD}'"

echo "DSN='postgres://greenlight:${DB_PASSWORD}@localhost/greenlight'" >> /etc/environment

apt install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
apt update
apt --yes install caddy

apt --yes -o Dpkg::Options::="--force-confnew" upgrade

echo "Script complete! Rebooting..."
reboot