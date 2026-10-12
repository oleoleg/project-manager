import os
from PIL import Image, ImageDraw

def create_installer_img(size):
    # Создаем холст высокого разрешения с прозрачным фоном
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)
    
    # Масштабируемый коэффициент
    scale = size / 256.0
    
    # 1. Рисуем изометрическую коробку (сервер/пакет ПО)
    # Задняя/темная грань коробки (для объема)
    draw.polygon([
        (128 * scale, 240 * scale),
        (230 * scale, 180 * scale),
        (230 * scale, 60 * scale),
        (128 * scale, 110 * scale)
    ], fill=(30, 41, 59, 255)) # Slate 800
    
    # Передняя главная грань коробки (ярко-синяя технологичная)
    draw.polygon([
        (26 * scale, 180 * scale),
        (128 * scale, 240 * scale),
        (128 * scale, 110 * scale),
        (26 * scale, 60 * scale)
    ], fill=(37, 99, 235, 255)) # Blue 600
    
    # Верхняя крышка коробки (светлый блик)
    draw.polygon([
        (26 * scale, 60 * scale),
        (128 * scale, 110 * scale),
        (230 * scale, 60 * scale),
        (128 * scale, 10 * scale)
    ], fill=(96, 165, 250, 255)) # Blue 400

    # 2. Наносим на переднюю грань элементы вашего бренда (Гант)
    # Полоса 1 (Зеленая)
    draw.polygon([
        (45 * scale, 95 * scale),
        (75 * scale, 110 * scale),
        (75 * scale, 118 * scale),
        (45 * scale, 103 * scale)
    ], fill=(16, 185, 129, 255))
    
    # Полоса 2 (Оранжевая)
    draw.polygon([
        (70 * scale, 123 * scale),
        (110 * scale, 143 * scale),
        (110 * scale, 151 * scale),
        (70 * scale, 131 * scale)
    ], fill=(249, 115, 22, 255))

    # 3. Добавляем символ инсталляции / раскрытия (Стрелка вниз на верхней крышке)
    # Рисуем стильную белую стрелку "скачивания/установки" на верхней плоскости
    draw.polygon([
        (118 * scale, 35 * scale),
        (138 * scale, 35 * scale),
        (138 * scale, 55 * scale),
        (148 * scale, 55 * scale),
        (128 * scale, 75 * scale),
        (108 * scale, 55 * scale),
        (118 * scale, 55 * scale)
    ], fill=(255, 255, 255, 230))

    return img

# Генерируем набор размеров для правильного .ico файла (Windows требует этого для инсталляторов)
sizes = [16, 32, 48, 64, 128, 256]
images = [create_installer_img(s) for s in sizes]

# Сохраняем в папку assets
script_dir = os.path.dirname(os.path.abspath(__file__))
icon_path = os.path.join(script_dir, "icon.ico")

# Первый элемент сохраняет остальные как слои внутри одного .ico
images[0].save(icon_path, format="ICO", append_images=images[1:], sizes=[(s, s) for s in sizes])
print("Мульти-размерная иконка для Inno Setup успешно создана в папке assets!")
