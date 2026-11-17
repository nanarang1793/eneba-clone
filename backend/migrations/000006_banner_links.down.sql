UPDATE banners
SET title = 'Big RPG Week',
    subtitle = 'Up to 60% off the biggest role-playing games',
    cta_label = 'See deals',
    cta_url = '/store?q=rpg&sort=price_asc'
WHERE cta_url = '/store?type=game&sort=popular';
