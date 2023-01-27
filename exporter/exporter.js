let cmd = mc.newCommand(
    'export',
    'Export blocks in a region to a JSON file'
)

cmd.mandatory(
    'start',
    ParamType.BlockPos
)

cmd.mandatory(
    'end',
    ParamType.BlockPos
)

cmd.overload(['start', 'end'])

cmd.setCallback((cmd, origin, output, results) => {
    logger.log('Exporting level data...')

    let startTimestamp = Date.now()

    let startPos = results['start']
    let endPos = results['end']

    if (startPos == undefined) {
        return output.error('Start position is null')
    }

    if (endPos == undefined) {
        return output.error('End position is null')
    }

    if (startPos.dim != endPos.dim) {
        return output.error('Start and end positions must be in the same dimension')
    }

    if (startPos.x > endPos.x || startPos.y > endPos.y || startPos.z > endPos.z) {
        return output.error('Start position must be less than end position')
    }

    // Make an empty 3D array
    let blocks = Array.from(
        Array(endPos.x - startPos.x),
        () => Array(endPos.y - startPos.y)
            .fill().map(
                () => Array(endPos.z - startPos.z).fill(0)
            )
    )

    logger.log(`Got ready in ${Date.now() - startTimestamp}ms`)

    // Fill the array with block IDs
    let dimid = startPos.dimid
    let count = 0
    let lastCount = 0
    let periodStartTimestamp = Date.now()
    for (let x = startPos.x; x < endPos.x; x++) {
        for (let y = startPos.y; y < endPos.y; y++) {
            for (let z = startPos.z; z < endPos.z; z++) {
                let blockObj = mc.getBlock(x, y, z, dimid)
                if (blockObj == null) {
                    return output.error(`Block at ${x}, ${y}, ${z} is null`)
                }
                blocks[x][y][z] = blockObj.id
                ++count
                let periodLength = Date.now() - periodStartTimestamp
                if (periodLength > 1000) {
                    logger.log(``)
                    logger.log(`Processed ${count} / ${blocks.length * blocks[0].length * blocks[0][0].length} blocks`)
                    logger.log(`Current speed: ${(count - lastCount) * 1000 / periodLength} blocks/s`)
                    logger.log(`Average speed: ${count * 1000 / (Date.now() - startTimestamp)} blocks/s`)
                    lastCount = count
                    periodStartTimestamp = Date.now()
                }
            }
        }
    }

    let jsonStr = JSON.stringify(blocks)
    File.mkdir('plugins/level_exporter')

    if (!File.writeTo('plugins/level_exporter/level_data.json', jsonStr)) {
        return output.error('Failed to write to plugins/level_exporter/level_data.json')
    }

    logger.log('Successfully exported level data to plugins/level_exporter/level_data.json')
    logger.log(`Took ${Date.now() - startTimestamp}ms`)

    return output.success('Successfully exported level data to plugins/level_exporter/level_data.json')
})

cmd.setup()