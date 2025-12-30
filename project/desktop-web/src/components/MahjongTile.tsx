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
        // 类型映射：wan=m(万), tong=p(饼), tiao=s(条)
        const typeMap: Record<'wan' | 'tiao' | 'tong', string> = {
            wan: 'm',
            tong: 'p',
            tiao: 's',
        };
        const suffix = typeMap[type];
        return `/src/assets/mahjong/${value}${suffix}.svg`;
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
