# Установить Mosquitto
sudo apt update
sudo apt install -y mosquitto mosquitto-clients

# Запустить и добавить в автозагрузку
sudo systemctl start mosquitto
sudo systemctl enable mosquitto

# Проверить статус
sudo systemctl status mosquitto
