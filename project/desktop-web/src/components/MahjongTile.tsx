import styles from './MahjongTile.module.css';

interface MahjongTileProps {
    type: 'wan' | 'tiao' | 'tong';
    value: 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9;
    selected?: boolean;
    onClick?: () => void;
    size?: 'small' | 'normal';
}

function MahjongTileComponent({ type, value, selected, onClick, size = 'normal' }: MahjongTileProps) {
    const getSvgPath = () => {
        return `/src/assets/mahjong/${type}-${value}.png`;
    };

    return (
        <div
            className={`${styles.tile} ${styles[size]} ${selected ? styles.selected : ''}`}
            onClick={onClick}
        >
            <img src={getSvgPath()} alt={`${type}-${value}`} className={styles.tileImage} />
        </div>
    );
}

export default MahjongTileComponent;
