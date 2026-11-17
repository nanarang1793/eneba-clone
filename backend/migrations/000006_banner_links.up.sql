-- The second banner linked to /store?q=rpg, but search only matches product
-- names and none contain "RPG", so the page was empty. Point it at
-- something that exists. (The seed generator was updated the same way.)
UPDATE banners
SET title = 'Bestselling games',
    subtitle = 'The games everyone is buying right now',
    cta_label = 'Shop bestsellers',
    cta_url = '/store?type=game&sort=popular'
WHERE cta_url = '/store?q=rpg&sort=price_asc';
