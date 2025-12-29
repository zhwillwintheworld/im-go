const fs = require('fs');
const { promisify } = require('util');
const exec = promisify(require('child_process').exec);

async function cropTiles() {
    const inputImage = 'src/assets/mahjong/image.png';
    const outputDir = 'src/assets/mahjong';

    // 检查输入文件是否存在
    if (!fs.existsSync(inputImage)) {
        console.error('输入文件不存在:', inputImage);
        process.exit(1);
    }

    // 获取图片信息
    const { stdout } = await exec(`identify -format "%wx%h" ${inputImage}`);
    const [width, height] = stdout.trim().split('x').map(Number);
    console.log(`图片尺寸: ${width}x${height}`);

    // 计算每张牌的尺寸（10列x4行）
    const cols = 10;
    const rows = 4;
    const tileWidth = Math.floor(width / cols);
    const tileHeight = Math.floor(height / rows);
    console.log(`每张牌尺寸: ${tileWidth}x${tileHeight}`);

    // 定义要提取的牌
    const tilesToExport = [
        { type: 'wan', row: 0, cols: Array.from({ length: 9 }, (_, i) => i) },   // 万牌：第0行，0-8列
        { type: 'tiao', row: 1, cols: Array.from({ length: 9 }, (_, i) => i) },  // 条牌：第1行，0-8列
        { type: 'tong', row: 2, cols: Array.from({ length: 9 }, (_, i) => i) },  // 筒牌：第2行，0-8列
    ];

    for (const { type, row, cols: colsRange } of tilesToExport) {
        for (const col of colsRange) {
            const value = col + 1;
            const x = col * tileWidth;
            const y = row * tileHeight;

            const outputFile = `${outputDir}/${type}-${value}.png`;

            // 使用 ImageMagick 裁剪
            const cropCmd = `convert ${inputImage} -crop ${tileWidth}x${tileHeight}+${x}+${y} ${outputFile}`;

            try {
                await exec(cropCmd);
                console.log(`Created ${outputFile}`);
            } catch (error) {
                console.error(`Error creating ${outputFile}:`, error.message);
            }
        }
    }

    console.log('完成！');
}

cropTiles().catch(console.error);
