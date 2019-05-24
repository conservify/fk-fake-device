@Library('conservify') _

conservifyProperties([
    pipelineTriggers([])
])

timestamps {
    node {
        stage ('git') {
            checkout scm
        }

        stage ('build') {
            sh "make clean all"
        }

        stage ('archive') {
            archiveArtifacts "fake-device"
        }
    }
}
